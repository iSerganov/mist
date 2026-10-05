#!/usr/bin/env python3
"""A CNN warden for the Mist harness.

Reads what `make harness MIST_HARNESS_EXPORT=dir` wrote, trains a small 1-D
CNN to tell the stego copy of a carrier from its clean ffmpeg copy, and
scores it the way the harness scores its own detectors: cross-validated by
lineage, one score per file, a 95% interval bootstrapped by carrier. A
single lineage falls back to carrier folds.
"""
import argparse
import json
import pathlib
import platform

import numpy as np
import torch
import torch.nn as nn

CLIP = 8


def load(directory):
    carriers = []
    for meta in sorted(directory.glob("*.json")):
        stem = meta.with_suffix("")
        clean = np.fromfile(f"{stem}.clean.i32", dtype="<i4")
        stego = np.fromfile(f"{stem}.stego.i32", dtype="<i4")
        identity = json.loads(meta.read_text())
        carriers.append({
            "id": identity["name"],
            "lineage": identity.get("lineage") or identity["name"],
            "clean": clean,
            "stego": stego,
        })
    return carriers


def assign_folds(carriers, n_folds, rng):
    """Keep every recording of one lineage in the same fold.

    One lineage cannot be split, so the split falls back to carriers.
    """
    lineages = [c["lineage"] for c in carriers]
    unique = list(dict.fromkeys(lineages))
    if len(unique) < 2:
        folds = np.arange(len(carriers)) % min(n_folds, len(carriers))
        rng.shuffle(folds)
        return folds
    order = np.array(unique)
    rng.shuffle(order)
    fold_of = {name: i % min(n_folds, len(order)) for i, name in enumerate(order)}
    return np.array([fold_of[name] for name in lineages])


def segments(values, length):
    n = len(values) // length
    return values[: n * length].reshape(n, length)


def features(batch):
    """Clipped first and second differences plus the value's parity."""
    v = batch.astype(np.int64)
    d1 = np.clip(v[:, 1:-1] - v[:, :-2], -CLIP, CLIP)
    d2 = np.clip(v[:, 2:] - 2 * v[:, 1:-1] + v[:, :-2], -CLIP, CLIP)
    parity = (v[:, 1:-1] & 1) * 2 - 1
    x = np.stack([d1 / CLIP, d2 / CLIP, parity], axis=1)
    return torch.from_numpy(x.astype(np.float32))


class Hybrid(nn.Module):
    """Waveform convolution with an attention pool. Not a transformer."""

    def __init__(self):
        super().__init__()
        self.conv = nn.Sequential(
            nn.Conv1d(3, 16, 5, padding=2), nn.ReLU(),
            nn.Conv1d(16, 16, 5, stride=2, padding=2), nn.ReLU(),
        )
        self.attn = nn.Linear(16, 1)
        self.out = nn.Linear(16, 1)

    def forward(self, x):
        h = self.conv(x)
        weight = torch.softmax(self.attn(h.transpose(1, 2)).squeeze(-1), dim=-1)
        return self.out((h * weight.unsqueeze(1)).sum(-1))


def waveform():
    def block(i, o, stride):
        return [nn.Conv1d(i, o, 5, stride, 2), nn.BatchNorm1d(o), nn.ReLU()]

    return nn.Sequential(
        *block(3, 16, 1), *block(16, 32, 2), *block(32, 64, 2), *block(64, 64, 2),
        nn.AdaptiveAvgPool1d(1), nn.Flatten(), nn.Linear(64, 1),
    )


def spectrogram():
    return nn.Sequential(
        nn.Conv2d(1, 8, 3, padding=1), nn.ReLU(), nn.MaxPool2d(2),
        nn.Conv2d(8, 16, 3, padding=1), nn.ReLU(),
        nn.AdaptiveAvgPool2d(1), nn.Flatten(), nn.Linear(16, 1),
    )


def build(arch):
    if arch == "spectrogram":
        return spectrogram()
    if arch == "hybrid":
        return Hybrid()
    # waveform and residue share this net. Residue means the integers are
    # codebook indices; the export already is, for a Vorbis run.
    return waveform()


def featurize(batch, arch):
    if arch == "spectrogram":
        v = torch.from_numpy(batch.astype(np.float32))
        spec = torch.stft(v, n_fft=128, hop_length=64, win_length=128, return_complex=True, center=False)
        return spec.abs().clamp_min(1e-6).log().unsqueeze(1)
    return features(batch)


def views(values, stereo):
    if not stereo or len(values) < 4:
        return [values]
    left, right = values[0::2], values[1::2]
    n = min(len(left), len(right))
    left, right = left[:n], right[:n]
    return [left, right, (left + right) // 2, (left - right) // 2]


def train(net, x, y, epochs, batch, rng, arch):
    opt = torch.optim.Adam(net.parameters(), lr=1e-3)
    loss_fn = nn.BCEWithLogitsLoss()
    net.train()
    for _ in range(epochs):
        order = rng.permutation(len(x))
        for i in range(0, len(order), batch):
            idx = order[i : i + batch]
            opt.zero_grad()
            loss = loss_fn(net(featurize(x[idx], arch)).squeeze(1), torch.from_numpy(y[idx]))
            loss.backward()
            opt.step()


def score_one(net, values, length, batch, arch):
    seg = segments(values, length)
    if len(seg) == 0:
        return 0.5
    net.eval()
    out = []
    with torch.no_grad():
        for i in range(0, len(seg), batch):
            out.append(net(featurize(seg[i : i + batch], arch)).squeeze(1).numpy())
    return float(np.concatenate(out).mean())


def score(net, values, length, batch, arch, stereo):
    return float(np.mean([score_one(net, part, length, batch, arch) for part in views(values, stereo)]))


def auc(pos, neg):
    pos, neg = np.asarray(pos), np.asarray(neg)
    return float((pos[:, None] > neg[None, :]).mean() + 0.5 * (pos[:, None] == neg[None, :]).mean())


def bootstrap(pos, neg, rounds, rng):
    n = len(pos)
    aucs = []
    for _ in range(rounds):
        pick = rng.integers(0, n, n)
        aucs.append(auc(np.asarray(pos)[pick], np.asarray(neg)[pick]))
    return float(np.percentile(aucs, 2.5)), float(np.percentile(aucs, 97.5))


def bag_auc(carriers, pos, neg):
    groups = {}
    for i, carrier in enumerate(carriers):
        groups.setdefault(carrier["lineage"], [[], []])
        groups[carrier["lineage"]][0].append(pos[i])
        groups[carrier["lineage"]][1].append(neg[i])
    return auc(
        [float(np.mean(v[0])) for v in groups.values()],
        [float(np.mean(v[1])) for v in groups.values()],
    )


def run_arch(carriers, args, rng, arch):
    folds = assign_folds(carriers, args.folds, rng)
    pos, neg = np.zeros(len(carriers)), np.zeros(len(carriers))
    for f in sorted(set(folds)):
        train_ids = [i for i in range(len(carriers)) if folds[i] != f]
        x = np.concatenate([segments(c[k], args.length) for i in train_ids
                            for c in [carriers[i]] for k in ("clean", "stego")])
        y = np.concatenate([np.full(len(segments(c[k], args.length)), label, dtype=np.float32)
                            for i in train_ids for c in [carriers[i]]
                            for k, label in (("clean", 0), ("stego", 1))])
        net = build(arch)
        train(net, x, y, args.epochs, args.batch, rng, arch)
        for i in range(len(carriers)):
            if folds[i] == f:
                neg[i] = score(net, carriers[i]["clean"], args.length, args.batch, arch, args.stereo)
                pos[i] = score(net, carriers[i]["stego"], args.length, args.batch, arch, args.stereo)
        print(f"  {arch} fold {f}: done", flush=True)
    lo, hi = bootstrap(pos, neg, args.rounds, rng)
    return {
        "architecture": arch,
        "files": len(carriers),
        "auc": auc(pos, neg),
        "bag_auc": bag_auc(carriers, pos, neg),
        "lo": lo,
        "hi": hi,
        "folds": args.folds,
        "epochs": args.epochs,
        "length": args.length,
        "batch": args.batch,
        "rounds": args.rounds,
        "seed": args.seed,
        "python": platform.python_version(),
        "numpy": np.__version__,
        "torch": torch.__version__,
        "scores": [
            {"carrier": carrier["id"], "clean": float(neg[i]), "stego": float(pos[i])}
            for i, carrier in enumerate(carriers)
        ],
    }


def run(directory, args, rng):
    carriers = load(directory)
    if len(carriers) < 2:
        return None
    names = ["waveform", "spectrogram", "hybrid", "residue"] if args.arch == "all" else [args.arch]
    primary = None
    held = []
    for arch in names:
        result = run_arch(carriers, args, rng, arch)
        if primary is None:
            primary = result
            continue
        held.append({
            "architecture": result["architecture"],
            "auc": result["auc"],
            "lo": result["lo"],
            "hi": result["hi"],
            "bag_auc": result["bag_auc"],
        })
    if held:
        primary["held_out"] = held
    return primary


def self_test(seed):
    rng = np.random.default_rng(seed)
    torch.manual_seed(seed)
    carriers = []
    for i in range(16):
        walk = np.cumsum(rng.normal(0, 1, 8192)).astype(np.int32)
        clean = walk & ~np.int32(1)
        stego = clean | rng.integers(0, 2, len(walk), dtype=np.int32)
        carriers.append({"id": f"c{i}", "lineage": f"l{i // 2}", "clean": clean, "stego": stego})

    class Args:
        pass

    args = Args()
    args.folds = 4
    args.epochs = 8
    args.length = 1024
    args.batch = 64
    args.rounds = 50
    args.seed = seed
    args.stereo = False
    result = run_arch(carriers, args, np.random.default_rng(seed), "waveform")
    print(f"self-test waveform AUC {result['auc']:.3f}", flush=True)
    if result["auc"] < 0.8:
        raise SystemExit(f"waveform positive control failed: AUC {result['auc']:.3f}")


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("export", nargs="?", type=pathlib.Path, help="directory MIST_HARNESS_EXPORT wrote")
    ap.add_argument("--out", type=pathlib.Path, help="cnn.json to write, in the harness output directory")
    ap.add_argument("--formats", nargs="*", help="export subdirectories to score; default all")
    ap.add_argument("--folds", type=int, default=5)
    ap.add_argument("--epochs", type=int, default=6)
    ap.add_argument("--length", type=int, default=8192)
    ap.add_argument("--batch", type=int, default=128)
    ap.add_argument("--rounds", type=int, default=1000)
    ap.add_argument("--seed", type=int, default=1)
    ap.add_argument("--arch", default="waveform", choices=["waveform", "spectrogram", "hybrid", "residue", "all"],
                    help="waveform is the default; all also reports spectrogram, hybrid and residue")
    ap.add_argument("--stereo", action="store_true",
                    help="score even/odd samples as left, right, mid and side; off by default")
    ap.add_argument("--self-test", action="store_true",
                    help="train the waveform net on synthetic LSB replacement and require AUC above 0.8")
    args = ap.parse_args()
    if args.self_test:
        self_test(args.seed)
        return
    if args.export is None or args.out is None:
        ap.error("export and --out are required unless --self-test is set")

    torch.manual_seed(args.seed)
    rng = np.random.default_rng(args.seed)
    results = json.loads(args.out.read_text()) if args.out.exists() else {}
    for directory in sorted(d for d in args.export.iterdir() if d.is_dir()):
        if args.formats and directory.name not in args.formats:
            continue
        print(directory.name, flush=True)
        if result := run(directory, args, rng):
            results[directory.name] = result
            print(f"  file AUC {result['auc']:.3f} ({result['lo']:.3f}-{result['hi']:.3f}) over {result['files']} carriers", flush=True)
            args.out.write_text(json.dumps(results, indent=2) + "\n")


if __name__ == "__main__":
    main()
