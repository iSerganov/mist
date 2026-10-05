#!/usr/bin/env python3
"""A CNN warden for the Mist harness.

Reads what `make harness MIST_HARNESS_EXPORT=dir` wrote, trains a small 1-D
CNN to tell the stego copy of a carrier from its clean ffmpeg copy, and
scores it the way the harness scores its own detectors: cross-validated by
carrier, one score per file, a 95% interval bootstrapped by carrier.
"""
import argparse
import json
import pathlib

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
        carriers.append((clean, stego))
    return carriers


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


def model():
    def block(i, o, stride):
        return [nn.Conv1d(i, o, 5, stride, 2), nn.BatchNorm1d(o), nn.ReLU()]

    return nn.Sequential(
        *block(3, 16, 1), *block(16, 32, 2), *block(32, 64, 2), *block(64, 64, 2),
        nn.AdaptiveAvgPool1d(1), nn.Flatten(), nn.Linear(64, 1),
    )


def train(net, x, y, epochs, batch, rng):
    opt = torch.optim.Adam(net.parameters(), lr=1e-3)
    loss_fn = nn.BCEWithLogitsLoss()
    net.train()
    for _ in range(epochs):
        order = rng.permutation(len(x))
        for i in range(0, len(order), batch):
            idx = order[i : i + batch]
            opt.zero_grad()
            loss = loss_fn(net(features(x[idx])).squeeze(1), torch.from_numpy(y[idx]))
            loss.backward()
            opt.step()


def score(net, values, length, batch):
    seg = segments(values, length)
    net.eval()
    out = []
    with torch.no_grad():
        for i in range(0, len(seg), batch):
            out.append(net(features(seg[i : i + batch])).squeeze(1).numpy())
    return float(np.concatenate(out).mean())


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


def run(directory, args, rng):
    carriers = load(directory)
    if len(carriers) < 2:
        return None
    folds = np.arange(len(carriers)) % min(args.folds, len(carriers))
    rng.shuffle(folds)
    pos, neg = np.zeros(len(carriers)), np.zeros(len(carriers))
    for f in sorted(set(folds)):
        train_ids = [i for i in range(len(carriers)) if folds[i] != f]
        x = np.concatenate([segments(c[k], args.length) for i in train_ids for c in [carriers[i]] for k in (0, 1)])
        y = np.concatenate([np.full(len(segments(c[k], args.length)), k, dtype=np.float32)
                            for i in train_ids for c in [carriers[i]] for k in (0, 1)])
        net = model()
        train(net, x, y, args.epochs, args.batch, rng)
        for i in range(len(carriers)):
            if folds[i] == f:
                neg[i] = score(net, carriers[i][0], args.length, args.batch)
                pos[i] = score(net, carriers[i][1], args.length, args.batch)
        print(f"  fold {f}: done", flush=True)
    lo, hi = bootstrap(pos, neg, args.rounds, rng)
    return {"files": len(carriers), "auc": auc(pos, neg), "lo": lo, "hi": hi, "epochs": args.epochs}


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("export", type=pathlib.Path, help="directory MIST_HARNESS_EXPORT wrote")
    ap.add_argument("--out", type=pathlib.Path, required=True, help="cnn.json to write, in the harness output directory")
    ap.add_argument("--formats", nargs="*", help="export subdirectories to score; default all")
    ap.add_argument("--folds", type=int, default=5)
    ap.add_argument("--epochs", type=int, default=6)
    ap.add_argument("--length", type=int, default=8192)
    ap.add_argument("--batch", type=int, default=128)
    ap.add_argument("--rounds", type=int, default=1000)
    ap.add_argument("--seed", type=int, default=1)
    args = ap.parse_args()

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
