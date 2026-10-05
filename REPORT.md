# Mist harness report

 perceptual metric: not installed

Every carrier is encoded three times: by the ffmpeg command line at its own defaults (the *clean* copy, which is what a warden without the original would compare against), by Mist's own encoder with nothing embedded (Mist's *own* re-encode), and by Mist with a hidden message (the *stego* copy). The report asks whether the stego copy differs from the clean one in plain properties, whether a detector can tell them apart, and how much worse it sounds. [How to read this report](#how-to-read-this-report) explains every number and threshold.

## Summary

| Format | Carriers | Looks like ffmpeg? | Hidden from detectors? | Audio quality |
|---|---|---|---|---|
| flac | 84 / 84 | ✅ matches ffmpeg on all 84 carriers | ⚠️ faint signal from chi-square (AUC 0.500) | ✅ inaudible: the added error is at least 94.13 dB below the music |
| wav/pcm_s16le | 84 / 84 | ✅ matches ffmpeg on all 84 carriers | ⚠️ faint signal from hcf-com (AUC 0.500) | ✅ inaudible: the added error is at least 82.56 dB below the music |
| ogg/vorbis | 84 / 84 | ❌ differs on 73 of 84 carriers: nominal bitrate | ⚠️ faint signal from spa (AUC 0.500) | ✅ Mist's own error sits 55.15 dB below the music; embedding costs 0.01 dB, within the 0.3 dB target |

## flac

Measured on 84 of 84 carriers.

### Does it look like a plain ffmpeg encode?

The ffmpeg column is ffmpeg at its own defaults. Ogg Vorbis keeps the source's quality, so it differs from ffmpeg's default q3 whenever the carrier maps to another level.

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 03 - Disposable Heroes.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 04 - No Remorse.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 06 - For Whom the Bell Tolls.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 07 - The Four Horsemen.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 08 - Fade to Black.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 09 - Seek & Destroy.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 10 - Whiplash.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 11 - Fight Fire with Fire.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| 14 - Motorbreath.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| ATB.06.20.2025/01 ATB 06-20-2025.mp3 | 6615983 | 6615983 / 6615983 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| ATB.06.20.2025/02 ATB 06-20-2025.mp3 | 6615983 | 6615983 / 6615983 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| ATB.06.20.2025/03 ATB 06-20-2025.mp3 | 6615983 | 6615983 / 6615983 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| ATB.06.20.2025/04 ATB 06-20-2025.mp3 | 6615983 | 6615983 / 6615983 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| Pillars_of_the_Sky.wav | 2880000 | 2880000 / 2880000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| Starlight_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| Stellar_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side1.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side2.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t01.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t02.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t03.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t04.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t05.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t06.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t07.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t08.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t09.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/01 Track01.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/02 Track02.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/03 Track03.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/04 Track04.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/05 Track05.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/06 Track06.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/07 Track07.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/08 Track08.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/09 Track09.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/10 Track10.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/11 Track11.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/12 Track12.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/13 Track13.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/14 Track14.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/15 Track15.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/16 Track16.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/17 Track17.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/18 Track18.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/19 Track19.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/20 Track20.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/21 Track21.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/22 Track22.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/23 Track23.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/24 Track24.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/25 Track25.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/26 Track26.flac | 6560316 | 6560316 / 6560316 | 20 / 20 | s16 / s16 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Adam Ellis - Broken (& Jo Cartwright).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Hurt of intention 2019 (& Neev Kennedy).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Right back.opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Alex Leavon - When I'm with you (& Julia Ross).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Ana Criado - In a thousand skies.opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Blue5even - Through the barricades (& Jo Cartwright).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Braulio Stefield - See ghosts (& Victoriya).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Costa - Always (& Cathy Burton).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Delta-S - Letting go (Ikerya Project remix) (& Kate Louise Smith).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Derek Ryan - After dark (ft Melissa R. Kaplan).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Drival - Saviour (& Michele C).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/F.G. Noise - Waiting for the thunder (Killing time) (& Lauren Ní Chasaide).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Kaimo K - Hold of you (Denis Kenzo remix).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Kaimo K - When you come home (Myde rework).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Karanda - Still got time (& Sarah Russell).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Limelght - Run & hide (ft Alina Renae).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Lost Witness - Sewn.opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Mhammed El Alami - Warriors (& Emma Horan).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Nicholas Gunn - Older (Costa remix) (ft Alina Renae).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Nitrous Oxide - Lower than the ground (& Sarah Russell).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Passenger 75 - Heartless (& Score).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Perpetual - Innocent (ft Fisher).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Raz Nitzan - Beyond time (Aurosonic remix) (& Ellie Lawson).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Ronski Speed - Beat alive (Denis Airwave remix) (& Sarah Lynn).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Rub!k - Everglow (& Sue McLaren).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Stargazers - Be here with me (ft Katty Heath).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Stargazers - Crystalize (& Fenna Day).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/The Blizzard - Always a stranger (Nitrous Oxide remix) (& Carol Lee).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/The City Never Sleeps - Addicted (NyTiGen remix) (& Summer Haze).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Whiteout - The part in-between (Wilderness & A-Line remix) (& One Half Bear).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s32 / s32 | 0 / 0 | — |

**Verdict:** ✅ matches ffmpeg on all 84 carriers.

### Can a detector tell?

Stego copy against the clean ffmpeg copy (for Ogg Vorbis, ffmpeg at the quality level Mist chose, so only the embedding differs).

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.503 (0.501–0.505) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.501 (0.500–0.502) | 0.504 (0.500–0.507) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.499 (0.499–0.500) | 0.498 (0.494–0.503) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.499) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.500 (0.499–0.501) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.507 (0.503–0.511) | 0.527 (0.512–0.546) | 0.001 (0.000–0.004) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.504 (0.502–0.506) | 0.519 (0.507–0.533) | 0.001 (0.000–0.002) | ⚠️ faint signal | ✅ hidden (0.500) |
| key-aware | the ephemeral key in the first frame's envelope, read with the public key alone | 0.500 (0.476–0.524) | 0.500 (0.476–0.524) | 0.000 (0.000–0.001) | ✅ chance | ✅ hidden (0.500) |

**Verdict:** ⚠️ faint signal from chi-square (AUC 0.500).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.503 (0.501–0.505) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.501 (0.500–0.502) | 0.504 (0.500–0.507) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.499 (0.499–0.500) | 0.498 (0.494–0.503) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.499) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.500 (0.499–0.501) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.507 (0.503–0.511) | 0.527 (0.512–0.546) | 0.001 (0.000–0.004) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.504 (0.502–0.506) | 0.519 (0.507–0.533) | 0.001 (0.000–0.002) | ⚠️ faint signal | ✅ hidden (0.500) |

### Does detection grow with audio?

File AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.

| Detector | First 40 chunks | First 240 chunks | Whole file |
|---|---|---|---|
| chi-square | 0.503 (0.501–0.505) | 0.503 (0.501–0.505) | 0.503 (0.501–0.505) |
| spa | 0.504 (0.500–0.508) | 0.503 (0.499–0.506) | 0.504 (0.500–0.507) |
| rs | 0.502 (0.496–0.507) | 0.499 (0.495–0.504) | 0.498 (0.494–0.503) |
| hcf-com | 0.500 (0.499–0.501) | 0.500 (0.499–0.501) | 0.500 (0.499–0.501) |
| classifier | 0.525 (0.510–0.543) | 0.527 (0.512–0.547) | 0.527 (0.512–0.546) |
| markov | 0.524 (0.510–0.543) | 0.518 (0.507–0.532) | 0.519 (0.507–0.533) |

### By kind of audio

File AUC (95%) against the clean ffmpeg copy, per corpus folder. The count is carriers; few carriers means a wide interval.

| Detector | (top level) (13) | ATB.06.20.2025 (4) | disc1 (2) | hmbb2015-03-18.sbd.flac24 (9) | sundaytimes-wind-in-the-willows (26) | va-female-vocal-trance-2019-opus-128 (30) |
|---|---|---|---|---|---|---|
| chi-square | 0.491 (0.481–0.500) | 0.531 (0.417–0.667) | 0.500 (0.500–0.500) | 0.519 (0.506–0.539) | 0.506 (0.498–0.516) | 0.516 (0.508–0.529) |
| spa | 0.509 (0.472–0.541) | 0.625 (0.600–0.667) | 0.500 (0.000–1.000) | 0.506 (0.457–0.562) | 0.506 (0.483–0.527) | 0.516 (0.504–0.531) |
| rs | 0.497 (0.430–0.557) | 0.562 (0.417–0.667) | 0.500 (0.000–1.000) | 0.469 (0.439–0.507) | 0.499 (0.479–0.523) | 0.491 (0.473–0.506) |
| hcf-com | 0.503 (0.494–0.513) | 0.469 (0.375–0.500) | 0.250 (0.000–0.333) | 0.543 (0.526–0.561) | 0.500 (0.492–0.511) | 0.499 (0.497–0.500) |
| classifier | 0.491 (0.459–0.515) | 0.625 (0.417–1.000) | 0.250 (0.000–0.333) | 0.753 (0.603–0.956) | 0.499 (0.491–0.506) | 0.942 (0.887–0.980) |
| markov | 0.497 (0.479–0.516) | 0.625 (0.417–1.000) | 0.500 (0.000–1.000) | 0.543 (0.519–0.562) | 0.494 (0.487–0.501) | 0.917 (0.846–0.974) |

### Does detection grow with the number of files?

AUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.

| Detector | 1 file(s) | 3 file(s) | 6 file(s) |
|---|---|---|---|
| chi-square | 0.485 | 0.485 | 0.486 |
| spa | 0.531 | 0.507 | 0.517 |
| rs | 0.520 | 0.508 | 0.511 |
| hcf-com | 0.492 | 0.487 | 0.482 |
| classifier | 0.543 | 0.537 | 0.532 |
| markov | 0.534 | 0.534 | 0.527 |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 67.37 dB | 36.40 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 87.74 dB | 36.40 dB | The same, with the message embedded |
| Embedding cost | 0.01 dB | 0.02 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +0% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 134.12 dB | 94.13 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ inaudible: the added error is at least 94.13 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 1587 | 128.61 dB | 128.59 dB | 0.02 dB | 152.00 dB |
| 03 - Disposable Heroes.mp3 | 1588 | 128.69 dB | 128.67 dB | 0.02 dB | 152.09 dB |
| 04 - No Remorse.mp3 | 1572 | 129.90 dB | 129.88 dB | 0.02 dB | 153.21 dB |
| 06 - For Whom the Bell Tolls.mp3 | 1545 | 127.73 dB | 127.71 dB | 0.02 dB | 151.19 dB |
| 07 - The Four Horsemen.mp3 | 1557 | 128.96 dB | 128.94 dB | 0.02 dB | 152.34 dB |
| 08 - Fade to Black.mp3 | 1535 | 127.90 dB | 127.88 dB | 0.02 dB | 151.38 dB |
| 09 - Seek & Destroy.mp3 | 1590 | 129.98 dB | 129.96 dB | 0.02 dB | 153.28 dB |
| 10 - Whiplash.mp3 | 1585 | 132.14 dB | 132.12 dB | 0.02 dB | 155.24 dB |
| 11 - Fight Fire with Fire.mp3 | 1510 | 127.69 dB | 127.67 dB | 0.02 dB | 151.17 dB |
| 14 - Motorbreath.mp3 | 1509 | 130.35 dB | 130.33 dB | 0.02 dB | 153.63 dB |
| ATB.06.20.2025/01 ATB 06-20-2025.mp3 | 1494 | 131.10 dB | 131.08 dB | 0.02 dB | 154.45 dB |
| ATB.06.20.2025/02 ATB 06-20-2025.mp3 | 1560 | 45.23 dB | 45.23 dB | 0.00 dB | 152.51 dB |
| ATB.06.20.2025/03 ATB 06-20-2025.mp3 | 1535 | 42.92 dB | 42.92 dB | 0.00 dB | 151.71 dB |
| ATB.06.20.2025/04 ATB 06-20-2025.mp3 | 1418 | 39.61 dB | 39.61 dB | 0.00 dB | 151.83 dB |
| Pillars_of_the_Sky.wav | 968 | — | 103.10 dB | — | 103.10 dB |
| Starlight_Ascent.wav | 1046 | — | 105.27 dB | — | 105.27 dB |
| Stellar_Ascent.wav | 978 | — | 105.99 dB | — | 105.99 dB |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side1.flac | 2296 | — | 135.19 dB | — | 135.19 dB |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side2.flac | 2241 | — | 130.65 dB | — | 130.65 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t01.flac | 2593 | — | 150.03 dB | — | 150.03 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t02.flac | 2693 | — | 150.20 dB | — | 150.20 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t03.flac | 2710 | — | 151.81 dB | — | 151.81 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t04.flac | 2584 | — | 148.67 dB | — | 148.67 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t05.flac | 2671 | — | 149.32 dB | — | 149.32 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t06.flac | 2679 | — | 149.45 dB | — | 149.45 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t07.flac | 2702 | — | 150.53 dB | — | 150.53 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t08.flac | 2688 | — | 150.71 dB | — | 150.71 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t09.flac | 2628 | — | 149.80 dB | — | 149.80 dB |
| sundaytimes-wind-in-the-willows/01 Track01.flac | 467 | — | 95.62 dB | — | 95.62 dB |
| sundaytimes-wind-in-the-willows/02 Track02.flac | 464 | — | 95.38 dB | — | 95.38 dB |
| sundaytimes-wind-in-the-willows/03 Track03.flac | 460 | — | 95.18 dB | — | 95.18 dB |
| sundaytimes-wind-in-the-willows/04 Track04.flac | 466 | — | 95.07 dB | — | 95.07 dB |
| sundaytimes-wind-in-the-willows/05 Track05.flac | 469 | — | 95.54 dB | — | 95.54 dB |
| sundaytimes-wind-in-the-willows/06 Track06.flac | 459 | — | 95.07 dB | — | 95.07 dB |
| sundaytimes-wind-in-the-willows/07 Track07.flac | 465 | — | 95.09 dB | — | 95.09 dB |
| sundaytimes-wind-in-the-willows/08 Track08.flac | 469 | — | 95.47 dB | — | 95.47 dB |
| sundaytimes-wind-in-the-willows/09 Track09.flac | 467 | — | 94.97 dB | — | 94.97 dB |
| sundaytimes-wind-in-the-willows/10 Track10.flac | 463 | — | 94.92 dB | — | 94.92 dB |
| sundaytimes-wind-in-the-willows/11 Track11.flac | 459 | — | 94.76 dB | — | 94.76 dB |
| sundaytimes-wind-in-the-willows/12 Track12.flac | 467 | — | 95.17 dB | — | 95.17 dB |
| sundaytimes-wind-in-the-willows/13 Track13.flac | 468 | — | 95.60 dB | — | 95.60 dB |
| sundaytimes-wind-in-the-willows/14 Track14.flac | 478 | — | 95.57 dB | — | 95.57 dB |
| sundaytimes-wind-in-the-willows/15 Track15.flac | 459 | — | 94.13 dB | — | 94.13 dB |
| sundaytimes-wind-in-the-willows/16 Track16.flac | 465 | — | 94.83 dB | — | 94.83 dB |
| sundaytimes-wind-in-the-willows/17 Track17.flac | 481 | — | 95.89 dB | — | 95.89 dB |
| sundaytimes-wind-in-the-willows/18 Track18.flac | 459 | — | 95.42 dB | — | 95.42 dB |
| sundaytimes-wind-in-the-willows/19 Track19.flac | 469 | — | 95.36 dB | — | 95.36 dB |
| sundaytimes-wind-in-the-willows/20 Track20.flac | 467 | — | 95.43 dB | — | 95.43 dB |
| sundaytimes-wind-in-the-willows/21 Track21.flac | 460 | — | 95.27 dB | — | 95.27 dB |
| sundaytimes-wind-in-the-willows/22 Track22.flac | 484 | — | 95.48 dB | — | 95.48 dB |
| sundaytimes-wind-in-the-willows/23 Track23.flac | 461 | — | 95.10 dB | — | 95.10 dB |
| sundaytimes-wind-in-the-willows/24 Track24.flac | 463 | — | 95.10 dB | — | 95.10 dB |
| sundaytimes-wind-in-the-willows/25 Track25.flac | 462 | — | 95.17 dB | — | 95.17 dB |
| sundaytimes-wind-in-the-willows/26 Track26.flac | 473 | — | 94.97 dB | — | 94.97 dB |
| va-female-vocal-trance-2019-opus-128/Adam Ellis - Broken (& Jo Cartwright).opus | 1878 | 46.75 dB | 46.75 dB | 0.00 dB | 157.75 dB |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Hurt of intention 2019 (& Neev Kennedy).opus | 1913 | 38.01 dB | 38.01 dB | 0.00 dB | 159.32 dB |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Right back.opus | 1789 | 47.54 dB | 47.54 dB | 0.00 dB | 156.81 dB |
| va-female-vocal-trance-2019-opus-128/Alex Leavon - When I'm with you (& Julia Ross).opus | 1908 | 36.40 dB | 36.40 dB | 0.00 dB | 157.85 dB |
| va-female-vocal-trance-2019-opus-128/Ana Criado - In a thousand skies.opus | 1914 | 45.28 dB | 45.28 dB | 0.00 dB | 157.52 dB |
| va-female-vocal-trance-2019-opus-128/Blue5even - Through the barricades (& Jo Cartwright).opus | 1814 | 61.12 dB | 61.12 dB | 0.00 dB | 154.30 dB |
| va-female-vocal-trance-2019-opus-128/Braulio Stefield - See ghosts (& Victoriya).opus | 1810 | 41.83 dB | 41.83 dB | 0.00 dB | 157.56 dB |
| va-female-vocal-trance-2019-opus-128/Costa - Always (& Cathy Burton).opus | 1895 | 43.91 dB | 43.91 dB | 0.00 dB | 158.09 dB |
| va-female-vocal-trance-2019-opus-128/Delta-S - Letting go (Ikerya Project remix) (& Kate Louise Smith).opus | 1890 | 44.97 dB | 44.97 dB | 0.00 dB | 158.31 dB |
| va-female-vocal-trance-2019-opus-128/Derek Ryan - After dark (ft Melissa R. Kaplan).opus | 1759 | 70.95 dB | 70.95 dB | 0.00 dB | 154.34 dB |
| va-female-vocal-trance-2019-opus-128/Drival - Saviour (& Michele C).opus | 1865 | 54.63 dB | 54.63 dB | 0.00 dB | 156.53 dB |
| va-female-vocal-trance-2019-opus-128/F.G. Noise - Waiting for the thunder (Killing time) (& Lauren Ní Chasaide).opus | 1900 | 43.76 dB | 43.76 dB | 0.00 dB | 157.62 dB |
| va-female-vocal-trance-2019-opus-128/Kaimo K - Hold of you (Denis Kenzo remix).opus | 1953 | 48.48 dB | 48.48 dB | 0.00 dB | 158.16 dB |
| va-female-vocal-trance-2019-opus-128/Kaimo K - When you come home (Myde rework).opus | 1928 | 39.69 dB | 39.69 dB | 0.00 dB | 159.41 dB |
| va-female-vocal-trance-2019-opus-128/Karanda - Still got time (& Sarah Russell).opus | 1814 | 54.41 dB | 54.41 dB | 0.00 dB | 157.52 dB |
| va-female-vocal-trance-2019-opus-128/Limelght - Run & hide (ft Alina Renae).opus | 1817 | 47.27 dB | 47.27 dB | 0.00 dB | 157.41 dB |
| va-female-vocal-trance-2019-opus-128/Lost Witness - Sewn.opus | 1875 | 43.03 dB | 43.03 dB | 0.00 dB | 156.58 dB |
| va-female-vocal-trance-2019-opus-128/Mhammed El Alami - Warriors (& Emma Horan).opus | 1892 | 46.45 dB | 46.45 dB | 0.00 dB | 158.60 dB |
| va-female-vocal-trance-2019-opus-128/Nicholas Gunn - Older (Costa remix) (ft Alina Renae).opus | 1814 | 50.75 dB | 50.75 dB | 0.00 dB | 155.22 dB |
| va-female-vocal-trance-2019-opus-128/Nitrous Oxide - Lower than the ground (& Sarah Russell).opus | 1897 | 40.55 dB | 40.55 dB | 0.00 dB | 158.54 dB |
| va-female-vocal-trance-2019-opus-128/Passenger 75 - Heartless (& Score).opus | 1807 | 43.62 dB | 43.62 dB | 0.00 dB | 157.65 dB |
| va-female-vocal-trance-2019-opus-128/Perpetual - Innocent (ft Fisher).opus | 1977 | 40.04 dB | 40.04 dB | 0.00 dB | 158.63 dB |
| va-female-vocal-trance-2019-opus-128/Raz Nitzan - Beyond time (Aurosonic remix) (& Ellie Lawson).opus | 1875 | 48.98 dB | 48.98 dB | 0.00 dB | 158.17 dB |
| va-female-vocal-trance-2019-opus-128/Ronski Speed - Beat alive (Denis Airwave remix) (& Sarah Lynn).opus | 1862 | 42.50 dB | 42.50 dB | 0.00 dB | 159.26 dB |
| va-female-vocal-trance-2019-opus-128/Rub!k - Everglow (& Sue McLaren).opus | 1909 | 46.15 dB | 46.15 dB | 0.00 dB | 157.59 dB |
| va-female-vocal-trance-2019-opus-128/Stargazers - Be here with me (ft Katty Heath).opus | 1944 | 46.99 dB | 46.99 dB | 0.00 dB | 157.77 dB |
| va-female-vocal-trance-2019-opus-128/Stargazers - Crystalize (& Fenna Day).opus | 1876 | 50.51 dB | 50.51 dB | 0.00 dB | 157.17 dB |
| va-female-vocal-trance-2019-opus-128/The Blizzard - Always a stranger (Nitrous Oxide remix) (& Carol Lee).opus | 1827 | 46.06 dB | 46.06 dB | 0.00 dB | 156.38 dB |
| va-female-vocal-trance-2019-opus-128/The City Never Sleeps - Addicted (NyTiGen remix) (& Summer Haze).opus | 1916 | 59.55 dB | 59.55 dB | 0.00 dB | 155.60 dB |
| va-female-vocal-trance-2019-opus-128/Whiteout - The part in-between (Wilderness & A-Line remix) (& One Half Bear).opus | 1804 | 43.41 dB | 43.41 dB | 0.00 dB | 158.32 dB |

## wav/pcm_s16le

Measured on 84 of 84 carriers.

### Does it look like a plain ffmpeg encode?

The ffmpeg column is ffmpeg at its own defaults. Ogg Vorbis keeps the source's quality, so it differs from ffmpeg's default q3 whenever the carrier maps to another level.

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 03 - Disposable Heroes.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 04 - No Remorse.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 06 - For Whom the Bell Tolls.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 07 - The Four Horsemen.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 08 - Fade to Black.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 09 - Seek & Destroy.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 10 - Whiplash.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 11 - Fight Fire with Fire.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| 14 - Motorbreath.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| ATB.06.20.2025/01 ATB 06-20-2025.mp3 | 6615983 | 6615983 / 6615983 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| ATB.06.20.2025/02 ATB 06-20-2025.mp3 | 6615983 | 6615983 / 6615983 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| ATB.06.20.2025/03 ATB 06-20-2025.mp3 | 6615983 | 6615983 / 6615983 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| ATB.06.20.2025/04 ATB 06-20-2025.mp3 | 6615983 | 6615983 / 6615983 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| Pillars_of_the_Sky.wav | 2880000 | 2880000 / 2880000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| Starlight_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| Stellar_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side1.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side2.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t01.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t02.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t03.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t04.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t05.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t06.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t07.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t08.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t09.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| sundaytimes-wind-in-the-willows/01 Track01.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/02 Track02.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/03 Track03.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/04 Track04.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/05 Track05.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/06 Track06.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/07 Track07.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/08 Track08.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/09 Track09.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/10 Track10.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/11 Track11.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/12 Track12.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/13 Track13.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/14 Track14.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/15 Track15.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/16 Track16.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/17 Track17.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/18 Track18.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/19 Track19.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/20 Track20.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/21 Track21.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/22 Track22.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/23 Track23.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/24 Track24.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/25 Track25.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/26 Track26.flac | 6560316 | 6560316 / 6560316 | 20 / 20 | s16 / s16 | 1411 / 1411 | — |
| va-female-vocal-trance-2019-opus-128/Adam Ellis - Broken (& Jo Cartwright).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Hurt of intention 2019 (& Neev Kennedy).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Right back.opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Alex Leavon - When I'm with you (& Julia Ross).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Ana Criado - In a thousand skies.opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Blue5even - Through the barricades (& Jo Cartwright).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Braulio Stefield - See ghosts (& Victoriya).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Costa - Always (& Cathy Burton).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Delta-S - Letting go (Ikerya Project remix) (& Kate Louise Smith).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Derek Ryan - After dark (ft Melissa R. Kaplan).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Drival - Saviour (& Michele C).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/F.G. Noise - Waiting for the thunder (Killing time) (& Lauren Ní Chasaide).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Kaimo K - Hold of you (Denis Kenzo remix).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Kaimo K - When you come home (Myde rework).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Karanda - Still got time (& Sarah Russell).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Limelght - Run & hide (ft Alina Renae).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Lost Witness - Sewn.opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Mhammed El Alami - Warriors (& Emma Horan).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Nicholas Gunn - Older (Costa remix) (ft Alina Renae).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Nitrous Oxide - Lower than the ground (& Sarah Russell).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Passenger 75 - Heartless (& Score).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Perpetual - Innocent (ft Fisher).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Raz Nitzan - Beyond time (Aurosonic remix) (& Ellie Lawson).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Ronski Speed - Beat alive (Denis Airwave remix) (& Sarah Lynn).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Rub!k - Everglow (& Sue McLaren).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Stargazers - Be here with me (ft Katty Heath).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Stargazers - Crystalize (& Fenna Day).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/The Blizzard - Always a stranger (Nitrous Oxide remix) (& Carol Lee).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/The City Never Sleeps - Addicted (NyTiGen remix) (& Summer Haze).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Whiteout - The part in-between (Wilderness & A-Line remix) (& One Half Bear).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |

**Verdict:** ✅ matches ffmpeg on all 84 carriers.

### Can a detector tell?

Stego copy against the clean ffmpeg copy (for Ogg Vorbis, ffmpeg at the quality level Mist chose, so only the embedding differs).

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.501 (0.499–0.503) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.499–0.501) | 0.499 (0.496–0.503) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.501) | 0.502 (0.497–0.506) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.502 (0.500–0.503) | 0.000 (0.000–0.000) | ⚠️ faint signal per file | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.501 (0.500–0.501) | 0.502 (0.500–0.505) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.500 (0.499–0.500) | 0.499 (0.497–0.501) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| key-aware | the ephemeral key in the first frame's envelope, read with the public key alone | 0.506 (0.470–0.542) | 0.506 (0.470–0.542) | 0.000 (0.000–0.003) | ✅ chance | ✅ hidden (0.500) |

**Verdict:** ⚠️ faint signal from hcf-com (AUC 0.500).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.501 (0.499–0.503) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.499–0.501) | 0.499 (0.496–0.503) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.501) | 0.502 (0.497–0.506) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.502 (0.500–0.503) | 0.000 (0.000–0.000) | ⚠️ faint signal per file | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.501 (0.500–0.501) | 0.502 (0.500–0.505) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.500 (0.499–0.500) | 0.499 (0.497–0.501) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |

### Does detection grow with audio?

File AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.

| Detector | First 40 chunks | First 240 chunks | Whole file |
|---|---|---|---|
| chi-square | 0.503 (0.500–0.508) | 0.501 (0.499–0.503) | 0.501 (0.499–0.503) |
| spa | 0.500 (0.495–0.505) | 0.500 (0.496–0.504) | 0.499 (0.496–0.503) |
| rs | 0.497 (0.490–0.503) | 0.502 (0.498–0.507) | 0.502 (0.497–0.506) |
| hcf-com | 0.500 (0.499–0.502) | 0.502 (0.500–0.503) | 0.502 (0.500–0.503) |
| classifier | 0.503 (0.500–0.505) | 0.504 (0.501–0.508) | 0.502 (0.500–0.505) |
| markov | 0.500 (0.498–0.502) | 0.500 (0.499–0.502) | 0.499 (0.497–0.501) |

### By kind of audio

File AUC (95%) against the clean ffmpeg copy, per corpus folder. The count is carriers; few carriers means a wide interval.

| Detector | (top level) (13) | ATB.06.20.2025 (4) | disc1 (2) | hmbb2015-03-18.sbd.flac24 (9) | sundaytimes-wind-in-the-willows (26) | va-female-vocal-trance-2019-opus-128 (30) |
|---|---|---|---|---|---|---|
| chi-square | 0.500 (0.485–0.515) | 0.469 (0.333–0.583) | 0.375 (0.000–0.500) | 0.500 (0.463–0.537) | 0.497 (0.481–0.508) | 0.513 (0.508–0.521) |
| spa | 0.521 (0.478–0.577) | 0.438 (0.333–0.583) | 0.750 (0.667–1.000) | 0.506 (0.467–0.546) | 0.491 (0.474–0.508) | 0.496 (0.479–0.511) |
| rs | 0.503 (0.422–0.585) | 0.438 (0.333–0.583) | 0.750 (0.667–1.000) | 0.519 (0.481–0.556) | 0.512 (0.493–0.542) | 0.501 (0.483–0.520) |
| hcf-com | 0.497 (0.473–0.516) | 0.562 (0.417–0.667) | 0.500 (0.000–1.000) | 0.494 (0.456–0.533) | 0.500 (0.493–0.508) | 0.510 (0.504–0.515) |
| classifier | 0.503 (0.484–0.524) | 0.500 (0.333–0.667) | 0.250 (0.000–0.333) | 0.469 (0.439–0.506) | 0.497 (0.486–0.506) | 0.683 (0.606–0.762) |
| markov | 0.515 (0.497–0.533) | 0.562 (0.417–0.667) | 0.250 (0.000–0.333) | 0.469 (0.439–0.506) | 0.494 (0.487–0.505) | 0.562 (0.528–0.603) |

### Does detection grow with the number of files?

AUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.

| Detector | 1 file(s) | 3 file(s) | 6 file(s) |
|---|---|---|---|
| chi-square | 0.548 | 0.493 | 0.509 |
| spa | 0.497 | 0.512 | 0.498 |
| rs | 0.488 | 0.507 | 0.489 |
| hcf-com | 0.475 | 0.480 | 0.479 |
| classifier | 0.526 | 0.520 | 0.505 |
| markov | 0.523 | 0.518 | 0.498 |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 59.08 dB | 36.40 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 71.89 dB | 36.40 dB | The same, with the message embedded |
| Embedding cost | 0.01 dB | 0.02 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +0% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 102.59 dB | 82.56 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ inaudible: the added error is at least 82.56 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 1411 | 79.81 dB | 79.79 dB | 0.02 dB | 103.83 dB |
| 03 - Disposable Heroes.mp3 | 1411 | 79.89 dB | 79.88 dB | 0.02 dB | 103.92 dB |
| 04 - No Remorse.mp3 | 1411 | 81.01 dB | 81.00 dB | 0.02 dB | 105.03 dB |
| 06 - For Whom the Bell Tolls.mp3 | 1411 | 78.99 dB | 78.97 dB | 0.02 dB | 103.03 dB |
| 07 - The Four Horsemen.mp3 | 1411 | 80.14 dB | 80.12 dB | 0.02 dB | 104.17 dB |
| 08 - Fade to Black.mp3 | 1411 | 79.18 dB | 79.16 dB | 0.02 dB | 103.20 dB |
| 09 - Seek & Destroy.mp3 | 1411 | 81.09 dB | 81.07 dB | 0.02 dB | 105.12 dB |
| 10 - Whiplash.mp3 | 1411 | 83.05 dB | 83.03 dB | 0.02 dB | 107.08 dB |
| 11 - Fight Fire with Fire.mp3 | 1411 | 78.97 dB | 78.96 dB | 0.02 dB | 103.00 dB |
| 14 - Motorbreath.mp3 | 1411 | 81.44 dB | 81.42 dB | 0.02 dB | 105.46 dB |
| ATB.06.20.2025/01 ATB 06-20-2025.mp3 | 1411 | 82.26 dB | 82.24 dB | 0.02 dB | 106.28 dB |
| ATB.06.20.2025/02 ATB 06-20-2025.mp3 | 1411 | 45.23 dB | 45.23 dB | 0.00 dB | 104.34 dB |
| ATB.06.20.2025/03 ATB 06-20-2025.mp3 | 1411 | 42.92 dB | 42.92 dB | 0.00 dB | 103.55 dB |
| ATB.06.20.2025/04 ATB 06-20-2025.mp3 | 1411 | 39.61 dB | 39.61 dB | 0.00 dB | 103.66 dB |
| Pillars_of_the_Sky.wav | 1536 | — | 103.10 dB | — | 103.10 dB |
| Starlight_Ascent.wav | 1536 | — | 105.27 dB | — | 105.27 dB |
| Stellar_Ascent.wav | 1536 | — | 106.00 dB | — | 106.00 dB |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side1.flac | 3072 | 63.02 dB | 63.00 dB | 0.02 dB | 87.10 dB |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side2.flac | 3072 | 58.48 dB | 58.47 dB | 0.02 dB | 82.56 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t01.flac | 3072 | 77.86 dB | 77.84 dB | 0.02 dB | 101.86 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t02.flac | 3072 | 78.03 dB | 78.01 dB | 0.02 dB | 102.03 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t03.flac | 3072 | 79.63 dB | 79.61 dB | 0.02 dB | 103.63 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t04.flac | 3072 | 76.51 dB | 76.49 dB | 0.02 dB | 100.51 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t05.flac | 3072 | 77.15 dB | 77.14 dB | 0.02 dB | 101.15 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t06.flac | 3072 | 77.28 dB | 77.26 dB | 0.02 dB | 101.28 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t07.flac | 3072 | 78.36 dB | 78.34 dB | 0.02 dB | 102.37 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t08.flac | 3072 | 78.54 dB | 78.52 dB | 0.02 dB | 102.54 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t09.flac | 3072 | 77.63 dB | 77.61 dB | 0.02 dB | 101.63 dB |
| sundaytimes-wind-in-the-willows/01 Track01.flac | 1411 | — | 95.60 dB | — | 95.60 dB |
| sundaytimes-wind-in-the-willows/02 Track02.flac | 1411 | — | 95.38 dB | — | 95.38 dB |
| sundaytimes-wind-in-the-willows/03 Track03.flac | 1411 | — | 95.18 dB | — | 95.18 dB |
| sundaytimes-wind-in-the-willows/04 Track04.flac | 1411 | — | 95.07 dB | — | 95.07 dB |
| sundaytimes-wind-in-the-willows/05 Track05.flac | 1411 | — | 95.53 dB | — | 95.53 dB |
| sundaytimes-wind-in-the-willows/06 Track06.flac | 1411 | — | 95.07 dB | — | 95.07 dB |
| sundaytimes-wind-in-the-willows/07 Track07.flac | 1411 | — | 95.09 dB | — | 95.09 dB |
| sundaytimes-wind-in-the-willows/08 Track08.flac | 1411 | — | 95.48 dB | — | 95.48 dB |
| sundaytimes-wind-in-the-willows/09 Track09.flac | 1411 | — | 94.97 dB | — | 94.97 dB |
| sundaytimes-wind-in-the-willows/10 Track10.flac | 1411 | — | 94.93 dB | — | 94.93 dB |
| sundaytimes-wind-in-the-willows/11 Track11.flac | 1411 | — | 94.75 dB | — | 94.75 dB |
| sundaytimes-wind-in-the-willows/12 Track12.flac | 1411 | — | 95.16 dB | — | 95.16 dB |
| sundaytimes-wind-in-the-willows/13 Track13.flac | 1411 | — | 95.60 dB | — | 95.60 dB |
| sundaytimes-wind-in-the-willows/14 Track14.flac | 1411 | — | 95.56 dB | — | 95.56 dB |
| sundaytimes-wind-in-the-willows/15 Track15.flac | 1411 | — | 94.12 dB | — | 94.12 dB |
| sundaytimes-wind-in-the-willows/16 Track16.flac | 1411 | — | 94.83 dB | — | 94.83 dB |
| sundaytimes-wind-in-the-willows/17 Track17.flac | 1411 | — | 95.89 dB | — | 95.89 dB |
| sundaytimes-wind-in-the-willows/18 Track18.flac | 1411 | — | 95.40 dB | — | 95.40 dB |
| sundaytimes-wind-in-the-willows/19 Track19.flac | 1411 | — | 95.36 dB | — | 95.36 dB |
| sundaytimes-wind-in-the-willows/20 Track20.flac | 1411 | — | 95.43 dB | — | 95.43 dB |
| sundaytimes-wind-in-the-willows/21 Track21.flac | 1411 | — | 95.27 dB | — | 95.27 dB |
| sundaytimes-wind-in-the-willows/22 Track22.flac | 1411 | — | 95.48 dB | — | 95.48 dB |
| sundaytimes-wind-in-the-willows/23 Track23.flac | 1411 | — | 95.12 dB | — | 95.12 dB |
| sundaytimes-wind-in-the-willows/24 Track24.flac | 1411 | — | 95.10 dB | — | 95.10 dB |
| sundaytimes-wind-in-the-willows/25 Track25.flac | 1411 | — | 95.17 dB | — | 95.17 dB |
| sundaytimes-wind-in-the-willows/26 Track26.flac | 1411 | — | 94.97 dB | — | 94.97 dB |
| va-female-vocal-trance-2019-opus-128/Adam Ellis - Broken (& Jo Cartwright).opus | 1536 | 46.74 dB | 46.74 dB | 0.00 dB | 109.59 dB |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Hurt of intention 2019 (& Neev Kennedy).opus | 1536 | 38.01 dB | 38.01 dB | 0.00 dB | 111.16 dB |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Right back.opus | 1536 | 47.54 dB | 47.54 dB | 0.00 dB | 108.65 dB |
| va-female-vocal-trance-2019-opus-128/Alex Leavon - When I'm with you (& Julia Ross).opus | 1536 | 36.40 dB | 36.40 dB | 0.00 dB | 109.69 dB |
| va-female-vocal-trance-2019-opus-128/Ana Criado - In a thousand skies.opus | 1536 | 45.28 dB | 45.28 dB | 0.00 dB | 109.34 dB |
| va-female-vocal-trance-2019-opus-128/Blue5even - Through the barricades (& Jo Cartwright).opus | 1536 | 61.08 dB | 61.08 dB | 0.00 dB | 106.13 dB |
| va-female-vocal-trance-2019-opus-128/Braulio Stefield - See ghosts (& Victoriya).opus | 1536 | 41.82 dB | 41.82 dB | 0.00 dB | 109.39 dB |
| va-female-vocal-trance-2019-opus-128/Costa - Always (& Cathy Burton).opus | 1536 | 43.91 dB | 43.91 dB | 0.00 dB | 109.93 dB |
| va-female-vocal-trance-2019-opus-128/Delta-S - Letting go (Ikerya Project remix) (& Kate Louise Smith).opus | 1536 | 44.97 dB | 44.97 dB | 0.00 dB | 110.14 dB |
| va-female-vocal-trance-2019-opus-128/Derek Ryan - After dark (ft Melissa R. Kaplan).opus | 1536 | 70.63 dB | 70.63 dB | 0.00 dB | 106.18 dB |
| va-female-vocal-trance-2019-opus-128/Drival - Saviour (& Michele C).opus | 1536 | 54.62 dB | 54.62 dB | 0.00 dB | 108.38 dB |
| va-female-vocal-trance-2019-opus-128/F.G. Noise - Waiting for the thunder (Killing time) (& Lauren Ní Chasaide).opus | 1536 | 43.76 dB | 43.76 dB | 0.00 dB | 109.46 dB |
| va-female-vocal-trance-2019-opus-128/Kaimo K - Hold of you (Denis Kenzo remix).opus | 1536 | 48.48 dB | 48.48 dB | 0.00 dB | 110.00 dB |
| va-female-vocal-trance-2019-opus-128/Kaimo K - When you come home (Myde rework).opus | 1536 | 39.69 dB | 39.69 dB | 0.00 dB | 111.24 dB |
| va-female-vocal-trance-2019-opus-128/Karanda - Still got time (& Sarah Russell).opus | 1536 | 54.40 dB | 54.40 dB | 0.00 dB | 109.36 dB |
| va-female-vocal-trance-2019-opus-128/Limelght - Run & hide (ft Alina Renae).opus | 1536 | 47.26 dB | 47.26 dB | 0.00 dB | 109.24 dB |
| va-female-vocal-trance-2019-opus-128/Lost Witness - Sewn.opus | 1536 | 43.03 dB | 43.03 dB | 0.00 dB | 108.41 dB |
| va-female-vocal-trance-2019-opus-128/Mhammed El Alami - Warriors (& Emma Horan).opus | 1536 | 46.45 dB | 46.45 dB | 0.00 dB | 110.42 dB |
| va-female-vocal-trance-2019-opus-128/Nicholas Gunn - Older (Costa remix) (ft Alina Renae).opus | 1536 | 50.75 dB | 50.75 dB | 0.00 dB | 107.05 dB |
| va-female-vocal-trance-2019-opus-128/Nitrous Oxide - Lower than the ground (& Sarah Russell).opus | 1536 | 40.54 dB | 40.54 dB | 0.00 dB | 110.38 dB |
| va-female-vocal-trance-2019-opus-128/Passenger 75 - Heartless (& Score).opus | 1536 | 43.61 dB | 43.61 dB | 0.00 dB | 109.49 dB |
| va-female-vocal-trance-2019-opus-128/Perpetual - Innocent (ft Fisher).opus | 1536 | 40.03 dB | 40.03 dB | 0.00 dB | 110.47 dB |
| va-female-vocal-trance-2019-opus-128/Raz Nitzan - Beyond time (Aurosonic remix) (& Ellie Lawson).opus | 1536 | 48.97 dB | 48.97 dB | 0.00 dB | 110.01 dB |
| va-female-vocal-trance-2019-opus-128/Ronski Speed - Beat alive (Denis Airwave remix) (& Sarah Lynn).opus | 1536 | 42.49 dB | 42.49 dB | 0.00 dB | 111.10 dB |
| va-female-vocal-trance-2019-opus-128/Rub!k - Everglow (& Sue McLaren).opus | 1536 | 46.15 dB | 46.15 dB | 0.00 dB | 109.42 dB |
| va-female-vocal-trance-2019-opus-128/Stargazers - Be here with me (ft Katty Heath).opus | 1536 | 46.99 dB | 46.99 dB | 0.00 dB | 109.60 dB |
| va-female-vocal-trance-2019-opus-128/Stargazers - Crystalize (& Fenna Day).opus | 1536 | 50.50 dB | 50.50 dB | 0.00 dB | 109.00 dB |
| va-female-vocal-trance-2019-opus-128/The Blizzard - Always a stranger (Nitrous Oxide remix) (& Carol Lee).opus | 1536 | 46.06 dB | 46.06 dB | 0.00 dB | 108.20 dB |
| va-female-vocal-trance-2019-opus-128/The City Never Sleeps - Addicted (NyTiGen remix) (& Summer Haze).opus | 1536 | 59.53 dB | 59.53 dB | 0.00 dB | 107.44 dB |
| va-female-vocal-trance-2019-opus-128/Whiteout - The part in-between (Wilderness & A-Line remix) (& One Half Bear).opus | 1536 | 43.41 dB | 43.41 dB | 0.00 dB | 110.15 dB |

## ogg/vorbis

Measured on 84 of 84 carriers.

### Does it look like a plain ffmpeg encode?

The ffmpeg column is ffmpeg at its own defaults. Ogg Vorbis keeps the source's quality, so it differs from ffmpeg's default q3 whenever the carrier maps to another level.

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 03 - Disposable Heroes.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 04 - No Remorse.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 06 - For Whom the Bell Tolls.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 07 - The Four Horsemen.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 08 - Fade to Black.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 09 - Seek & Destroy.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 10 - Whiplash.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 11 - Fight Fire with Fire.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| 14 - Motorbreath.mp3 | 6615407 | 6615407 / 6615407 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| ATB.06.20.2025/01 ATB 06-20-2025.mp3 | 6615983 | 6615983 / 6615983 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| ATB.06.20.2025/02 ATB 06-20-2025.mp3 | 6615983 | 6615983 / 6615983 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| ATB.06.20.2025/03 ATB 06-20-2025.mp3 | 6615983 | 6615983 / 6615983 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| ATB.06.20.2025/04 ATB 06-20-2025.mp3 | 6615983 | 6615983 / 6615983 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| Pillars_of_the_Sky.wav | 2880000 | 2880000 / 2880000 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| Starlight_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| Stellar_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side1.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side2.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t01.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t02.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t03.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t04.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t05.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t06.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t07.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t08.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t09.flac | 14401536 | 14401536 / 14401536 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| sundaytimes-wind-in-the-willows/01 Track01.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/02 Track02.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/03 Track03.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/04 Track04.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/05 Track05.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/06 Track06.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/07 Track07.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/08 Track08.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/09 Track09.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/10 Track10.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/11 Track11.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/12 Track12.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/13 Track13.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/14 Track14.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/15 Track15.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/16 Track16.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/17 Track17.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/18 Track18.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/19 Track19.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/20 Track20.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/21 Track21.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/22 Track22.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/23 Track23.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/24 Track24.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/25 Track25.flac | 6615040 | 6615040 / 6615040 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/26 Track26.flac | 6560316 | 6560316 / 6560316 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Adam Ellis - Broken (& Jo Cartwright).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Hurt of intention 2019 (& Neev Kennedy).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Right back.opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Alex Leavon - When I'm with you (& Julia Ross).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Ana Criado - In a thousand skies.opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Blue5even - Through the barricades (& Jo Cartwright).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Braulio Stefield - See ghosts (& Victoriya).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Costa - Always (& Cathy Burton).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Delta-S - Letting go (Ikerya Project remix) (& Kate Louise Smith).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Derek Ryan - After dark (ft Melissa R. Kaplan).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Drival - Saviour (& Michele C).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/F.G. Noise - Waiting for the thunder (Killing time) (& Lauren Ní Chasaide).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Kaimo K - Hold of you (Denis Kenzo remix).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Kaimo K - When you come home (Myde rework).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Karanda - Still got time (& Sarah Russell).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Limelght - Run & hide (ft Alina Renae).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Lost Witness - Sewn.opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Mhammed El Alami - Warriors (& Emma Horan).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Nicholas Gunn - Older (Costa remix) (ft Alina Renae).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Nitrous Oxide - Lower than the ground (& Sarah Russell).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Passenger 75 - Heartless (& Score).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Perpetual - Innocent (ft Fisher).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Raz Nitzan - Beyond time (Aurosonic remix) (& Ellie Lawson).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Ronski Speed - Beat alive (Denis Airwave remix) (& Sarah Lynn).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Rub!k - Everglow (& Sue McLaren).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Stargazers - Be here with me (ft Katty Heath).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Stargazers - Crystalize (& Fenna Day).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/The Blizzard - Always a stranger (Nitrous Oxide remix) (& Carol Lee).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/The City Never Sleeps - Addicted (NyTiGen remix) (& Summer Haze).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Whiteout - The part in-between (Wilderness & A-Line remix) (& One Half Bear).opus | 7200648 | 7200648 / 7200648 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |

**Verdict:** ❌ differs on 73 of 84 carriers: nominal bitrate.

### Can a detector tell?

Stego copy against the clean ffmpeg copy (for Ogg Vorbis, ffmpeg at the quality level Mist chose, so only the embedding differs).

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.505 (0.500–0.516) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.499–0.501) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.501 (0.501–0.501) | 0.507 (0.506–0.509) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.498 (0.498–0.499) | 0.493 (0.492–0.494) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.497 (0.496–0.497) | 0.490 (0.487–0.492) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| key-aware | the ephemeral key in the first frame's envelope, read with the public key alone | 0.506 (0.488–0.524) | 0.506 (0.488–0.524) | 0.000 (0.000–0.001) | ✅ chance | ✅ hidden (0.500) |

**Verdict:** ⚠️ faint signal from spa (AUC 0.500).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.505 (0.500–0.516) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.499–0.501) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.501 (0.501–0.501) | 0.507 (0.506–0.509) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.498 (0.498–0.499) | 0.493 (0.492–0.494) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.497 (0.496–0.497) | 0.490 (0.487–0.492) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |

### Does detection grow with audio?

File AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.

| Detector | First 40 chunks | Whole file |
|---|---|---|
| chi-square | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) |
| spa | 0.505 (0.500–0.516) | 0.505 (0.500–0.516) |
| rs | 0.500 (0.499–0.501) | 0.500 (0.499–0.501) |
| hcf-com | 0.508 (0.506–0.510) | 0.507 (0.506–0.509) |
| classifier | 0.491 (0.489–0.493) | 0.493 (0.492–0.494) |
| markov | 0.490 (0.486–0.492) | 0.490 (0.487–0.492) |

### By kind of audio

File AUC (95%) against the clean ffmpeg copy, per corpus folder. The count is carriers; few carriers means a wide interval.

| Detector | (top level) (13) | ATB.06.20.2025 (4) | disc1 (2) | hmbb2015-03-18.sbd.flac24 (9) | sundaytimes-wind-in-the-willows (26) | va-female-vocal-trance-2019-opus-128 (30) |
|---|---|---|---|---|---|---|
| chi-square | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.498–0.502) |
| spa | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.512 (0.498–0.538) |
| rs | 0.497 (0.491–0.500) | 0.469 (0.375–0.500) | 0.500 (0.000–1.000) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.503 (0.499–0.506) |
| hcf-com | 0.544 (0.537–0.577) | 0.625 (0.600–0.667) | 0.250 (0.000–0.333) | 0.556 (0.551–0.565) | 0.525 (0.519–0.537) | 0.518 (0.514–0.526) |
| classifier | 0.467 (0.435–0.491) | 0.375 (0.333–0.400) | 0.750 (0.667–1.000) | 0.556 (0.551–0.565) | 0.473 (0.457–0.481) | 0.479 (0.470–0.483) |
| markov | 0.456 (0.432–0.463) | 0.375 (0.333–0.400) | 0.750 (0.667–1.000) | 0.469 (0.439–0.506) | 0.487 (0.476–0.495) | 0.471 (0.457–0.480) |

### Does detection grow with the number of files?

AUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.

| Detector | 1 file(s) | 3 file(s) | 6 file(s) |
|---|---|---|---|
| chi-square | 0.505 | 0.496 | 0.498 |
| spa | 0.485 | 0.519 | 0.521 |
| rs | 0.476 | 0.501 | 0.519 |
| hcf-com | 0.527 | 0.487 | 0.507 |
| classifier | 0.511 | 0.502 | 0.493 |
| markov | 0.507 | 0.502 | 0.491 |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 27.80 dB | 24.41 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 27.79 dB | 24.39 dB | The same, with the message embedded |
| Embedding cost | 0.01 dB | 0.02 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +0% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 55.15 dB | 49.04 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ Mist's own error sits 55.15 dB below the music; embedding costs 0.01 dB, within the 0.3 dB target.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 228 | 27.80 dB | 27.79 dB | 0.01 dB | 56.54 dB |
| 03 - Disposable Heroes.mp3 | 225 | 28.23 dB | 28.22 dB | 0.01 dB | 56.58 dB |
| 04 - No Remorse.mp3 | 220 | 29.36 dB | 29.36 dB | 0.00 dB | 60.03 dB |
| 06 - For Whom the Bell Tolls.mp3 | 230 | 28.57 dB | 28.57 dB | 0.00 dB | 60.34 dB |
| 07 - The Four Horsemen.mp3 | 222 | 28.53 dB | 28.53 dB | 0.00 dB | 60.04 dB |
| 08 - Fade to Black.mp3 | 225 | 29.77 dB | 29.77 dB | 0.00 dB | 59.71 dB |
| 09 - Seek & Destroy.mp3 | 225 | 28.71 dB | 28.70 dB | 0.00 dB | 58.44 dB |
| 10 - Whiplash.mp3 | 234 | 29.93 dB | 29.93 dB | 0.00 dB | 60.42 dB |
| 11 - Fight Fire with Fire.mp3 | 191 | 26.93 dB | 26.93 dB | 0.01 dB | 55.16 dB |
| 14 - Motorbreath.mp3 | 190 | 29.51 dB | 29.50 dB | 0.01 dB | 56.46 dB |
| ATB.06.20.2025/01 ATB 06-20-2025.mp3 | 273 | 32.77 dB | 32.76 dB | 0.01 dB | 59.01 dB |
| ATB.06.20.2025/02 ATB 06-20-2025.mp3 | 272 | 27.88 dB | 27.87 dB | 0.01 dB | 57.13 dB |
| ATB.06.20.2025/03 ATB 06-20-2025.mp3 | 266 | 32.19 dB | 32.19 dB | 0.01 dB | 59.97 dB |
| ATB.06.20.2025/04 ATB 06-20-2025.mp3 | 253 | 31.54 dB | 31.53 dB | 0.01 dB | 61.17 dB |
| Pillars_of_the_Sky.wav | 262 | 26.02 dB | 26.01 dB | 0.01 dB | 52.10 dB |
| Starlight_Ascent.wav | 282 | 26.09 dB | 26.08 dB | 0.01 dB | 53.00 dB |
| Stellar_Ascent.wav | 261 | 27.80 dB | 27.79 dB | 0.01 dB | 54.28 dB |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side1.flac | 321 | 30.20 dB | 30.20 dB | 0.00 dB | 70.96 dB |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side2.flac | 286 | 30.66 dB | 30.66 dB | 0.00 dB | 74.36 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t01.flac | 262 | 29.54 dB | 29.54 dB | 0.00 dB | 60.38 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t02.flac | 211 | 28.37 dB | 28.36 dB | 0.00 dB | 58.06 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t03.flac | 260 | 30.13 dB | 30.12 dB | 0.01 dB | 58.05 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t04.flac | 224 | 30.25 dB | 30.25 dB | 0.00 dB | 61.49 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t05.flac | 264 | 28.97 dB | 28.97 dB | 0.00 dB | 58.79 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t06.flac | 226 | 27.74 dB | 27.73 dB | 0.01 dB | 56.44 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t07.flac | 291 | 27.84 dB | 27.83 dB | 0.01 dB | 56.11 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t08.flac | 223 | 26.71 dB | 26.71 dB | 0.01 dB | 56.20 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t09.flac | 215 | 28.10 dB | 28.10 dB | 0.00 dB | 58.14 dB |
| sundaytimes-wind-in-the-willows/01 Track01.flac | 233 | 28.57 dB | 28.56 dB | 0.01 dB | 54.33 dB |
| sundaytimes-wind-in-the-willows/02 Track02.flac | 237 | 29.21 dB | 29.20 dB | 0.01 dB | 56.46 dB |
| sundaytimes-wind-in-the-willows/03 Track03.flac | 238 | 29.53 dB | 29.52 dB | 0.01 dB | 55.93 dB |
| sundaytimes-wind-in-the-willows/04 Track04.flac | 246 | 29.51 dB | 29.51 dB | 0.01 dB | 57.17 dB |
| sundaytimes-wind-in-the-willows/05 Track05.flac | 227 | 28.90 dB | 28.89 dB | 0.01 dB | 55.02 dB |
| sundaytimes-wind-in-the-willows/06 Track06.flac | 244 | 29.50 dB | 29.49 dB | 0.01 dB | 57.13 dB |
| sundaytimes-wind-in-the-willows/07 Track07.flac | 239 | 28.93 dB | 28.92 dB | 0.01 dB | 56.07 dB |
| sundaytimes-wind-in-the-willows/08 Track08.flac | 228 | 28.85 dB | 28.84 dB | 0.01 dB | 54.27 dB |
| sundaytimes-wind-in-the-willows/09 Track09.flac | 246 | 28.61 dB | 28.60 dB | 0.01 dB | 56.49 dB |
| sundaytimes-wind-in-the-willows/10 Track10.flac | 249 | 29.30 dB | 29.29 dB | 0.01 dB | 55.68 dB |
| sundaytimes-wind-in-the-willows/11 Track11.flac | 250 | 29.61 dB | 29.60 dB | 0.01 dB | 57.02 dB |
| sundaytimes-wind-in-the-willows/12 Track12.flac | 245 | 28.95 dB | 28.94 dB | 0.01 dB | 55.94 dB |
| sundaytimes-wind-in-the-willows/13 Track13.flac | 234 | 28.87 dB | 28.86 dB | 0.01 dB | 54.93 dB |
| sundaytimes-wind-in-the-willows/14 Track14.flac | 241 | 28.03 dB | 28.02 dB | 0.01 dB | 54.30 dB |
| sundaytimes-wind-in-the-willows/15 Track15.flac | 234 | 28.96 dB | 28.95 dB | 0.01 dB | 54.51 dB |
| sundaytimes-wind-in-the-willows/16 Track16.flac | 244 | 29.63 dB | 29.62 dB | 0.01 dB | 56.64 dB |
| sundaytimes-wind-in-the-willows/17 Track17.flac | 242 | 27.81 dB | 27.80 dB | 0.01 dB | 55.97 dB |
| sundaytimes-wind-in-the-willows/18 Track18.flac | 243 | 29.22 dB | 29.21 dB | 0.01 dB | 56.62 dB |
| sundaytimes-wind-in-the-willows/19 Track19.flac | 247 | 29.18 dB | 29.18 dB | 0.01 dB | 56.36 dB |
| sundaytimes-wind-in-the-willows/20 Track20.flac | 236 | 29.09 dB | 29.07 dB | 0.01 dB | 56.09 dB |
| sundaytimes-wind-in-the-willows/21 Track21.flac | 232 | 29.34 dB | 29.33 dB | 0.01 dB | 55.87 dB |
| sundaytimes-wind-in-the-willows/22 Track22.flac | 239 | 27.05 dB | 27.04 dB | 0.01 dB | 52.91 dB |
| sundaytimes-wind-in-the-willows/23 Track23.flac | 242 | 28.48 dB | 28.47 dB | 0.01 dB | 54.56 dB |
| sundaytimes-wind-in-the-willows/24 Track24.flac | 230 | 28.64 dB | 28.63 dB | 0.01 dB | 56.64 dB |
| sundaytimes-wind-in-the-willows/25 Track25.flac | 230 | 29.36 dB | 29.35 dB | 0.01 dB | 58.09 dB |
| sundaytimes-wind-in-the-willows/26 Track26.flac | 233 | 27.31 dB | 27.30 dB | 0.01 dB | 53.13 dB |
| va-female-vocal-trance-2019-opus-128/Adam Ellis - Broken (& Jo Cartwright).opus | 264 | 26.74 dB | 26.72 dB | 0.01 dB | 52.40 dB |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Hurt of intention 2019 (& Neev Kennedy).opus | 250 | 26.99 dB | 26.98 dB | 0.01 dB | 52.81 dB |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Right back.opus | 262 | 26.24 dB | 26.22 dB | 0.02 dB | 50.33 dB |
| va-female-vocal-trance-2019-opus-128/Alex Leavon - When I'm with you (& Julia Ross).opus | 254 | 25.28 dB | 25.27 dB | 0.01 dB | 49.98 dB |
| va-female-vocal-trance-2019-opus-128/Ana Criado - In a thousand skies.opus | 247 | 24.65 dB | 24.64 dB | 0.01 dB | 49.84 dB |
| va-female-vocal-trance-2019-opus-128/Blue5even - Through the barricades (& Jo Cartwright).opus | 236 | 25.40 dB | 25.39 dB | 0.01 dB | 50.13 dB |
| va-female-vocal-trance-2019-opus-128/Braulio Stefield - See ghosts (& Victoriya).opus | 242 | 25.26 dB | 25.24 dB | 0.01 dB | 49.99 dB |
| va-female-vocal-trance-2019-opus-128/Costa - Always (& Cathy Burton).opus | 245 | 26.73 dB | 26.71 dB | 0.01 dB | 52.89 dB |
| va-female-vocal-trance-2019-opus-128/Delta-S - Letting go (Ikerya Project remix) (& Kate Louise Smith).opus | 252 | 26.10 dB | 26.09 dB | 0.01 dB | 52.53 dB |
| va-female-vocal-trance-2019-opus-128/Derek Ryan - After dark (ft Melissa R. Kaplan).opus | 231 | 25.79 dB | 25.78 dB | 0.01 dB | 52.34 dB |
| va-female-vocal-trance-2019-opus-128/Drival - Saviour (& Michele C).opus | 248 | 25.58 dB | 25.57 dB | 0.01 dB | 50.59 dB |
| va-female-vocal-trance-2019-opus-128/F.G. Noise - Waiting for the thunder (Killing time) (& Lauren Ní Chasaide).opus | 253 | 25.79 dB | 25.77 dB | 0.01 dB | 51.29 dB |
| va-female-vocal-trance-2019-opus-128/Kaimo K - Hold of you (Denis Kenzo remix).opus | 259 | 25.45 dB | 25.44 dB | 0.01 dB | 50.91 dB |
| va-female-vocal-trance-2019-opus-128/Kaimo K - When you come home (Myde rework).opus | 266 | 25.99 dB | 25.97 dB | 0.01 dB | 50.94 dB |
| va-female-vocal-trance-2019-opus-128/Karanda - Still got time (& Sarah Russell).opus | 233 | 26.13 dB | 26.12 dB | 0.01 dB | 51.92 dB |
| va-female-vocal-trance-2019-opus-128/Limelght - Run & hide (ft Alina Renae).opus | 250 | 27.27 dB | 27.26 dB | 0.01 dB | 53.31 dB |
| va-female-vocal-trance-2019-opus-128/Lost Witness - Sewn.opus | 244 | 24.74 dB | 24.73 dB | 0.01 dB | 49.39 dB |
| va-female-vocal-trance-2019-opus-128/Mhammed El Alami - Warriors (& Emma Horan).opus | 246 | 25.81 dB | 25.79 dB | 0.02 dB | 50.24 dB |
| va-female-vocal-trance-2019-opus-128/Nicholas Gunn - Older (Costa remix) (ft Alina Renae).opus | 250 | 25.34 dB | 25.33 dB | 0.01 dB | 50.33 dB |
| va-female-vocal-trance-2019-opus-128/Nitrous Oxide - Lower than the ground (& Sarah Russell).opus | 245 | 25.80 dB | 25.78 dB | 0.01 dB | 51.52 dB |
| va-female-vocal-trance-2019-opus-128/Passenger 75 - Heartless (& Score).opus | 258 | 25.39 dB | 25.38 dB | 0.01 dB | 50.20 dB |
| va-female-vocal-trance-2019-opus-128/Perpetual - Innocent (ft Fisher).opus | 271 | 24.71 dB | 24.69 dB | 0.02 dB | 49.04 dB |
| va-female-vocal-trance-2019-opus-128/Raz Nitzan - Beyond time (Aurosonic remix) (& Ellie Lawson).opus | 253 | 25.90 dB | 25.88 dB | 0.01 dB | 50.86 dB |
| va-female-vocal-trance-2019-opus-128/Ronski Speed - Beat alive (Denis Airwave remix) (& Sarah Lynn).opus | 248 | 27.41 dB | 27.40 dB | 0.01 dB | 54.25 dB |
| va-female-vocal-trance-2019-opus-128/Rub!k - Everglow (& Sue McLaren).opus | 246 | 24.52 dB | 24.51 dB | 0.01 dB | 49.86 dB |
| va-female-vocal-trance-2019-opus-128/Stargazers - Be here with me (ft Katty Heath).opus | 258 | 25.48 dB | 25.46 dB | 0.02 dB | 49.92 dB |
| va-female-vocal-trance-2019-opus-128/Stargazers - Crystalize (& Fenna Day).opus | 243 | 24.93 dB | 24.92 dB | 0.01 dB | 50.45 dB |
| va-female-vocal-trance-2019-opus-128/The Blizzard - Always a stranger (Nitrous Oxide remix) (& Carol Lee).opus | 246 | 26.04 dB | 26.03 dB | 0.01 dB | 52.09 dB |
| va-female-vocal-trance-2019-opus-128/The City Never Sleeps - Addicted (NyTiGen remix) (& Summer Haze).opus | 258 | 24.41 dB | 24.39 dB | 0.01 dB | 49.80 dB |
| va-female-vocal-trance-2019-opus-128/Whiteout - The part in-between (Wilderness & A-Line remix) (& One Half Bear).opus | 268 | 28.51 dB | 28.50 dB | 0.01 dB | 54.89 dB |

## How to read this report

### Plain properties

Before any statistics, a file shows how long it is, whether it ends in digital zeros, what sample format it
decodes to and what bitrate its header claims. The **ffmpeg** column is the same carrier encoded by the ffmpeg
command line at its own defaults, keeping only the first audio stream. Any difference from it tells Mist's output
apart from an ordinary file outright, however well the embedding hides, so every carrier must match.

### Detectability

**AUC** is the chance that a detector, shown one clean and one stego sample, picks the stego one. The main table
compares against the ffmpeg copy, the cover a warden would actually have. **The embedding alone** compares against
Mist's own re-encode, so it isolates the embedded changes from every other way Mist's pipeline differs from ffmpeg. 0.5 is a coin
flip, which is the goal; 1.0 means it is caught every time. A value well below 0.5 is a detection too: the
detector is right, just with its sign flipped. Ogg Vorbis is scored on the residues Mist may change, lossless
formats on the decoded samples.

Each carrier is cut into chunks of 65,536 values and every chunk is scored: that is the **chunk AUC**. The
**file AUC** averages each carrier's chunk scores into one score per file, which is what a warden holding whole
files would use; a bias too slight to show in one chunk can add up over a file. The **95% intervals** are
bootstrapped by carrier: each round redraws whole carriers, with all their chunks, because chunks of one track are
not independent. Their width reflects how many carriers there are, not how many chunks. The verdict is the worse
of the chunk and file verdicts.

**ε ≥** is what the file AUC proves about Mist in Cachin's sense, where a scheme is ε-secure when the relative
entropy between clean and stego files is at most ε. A detector's |AUC − ½| is at most the total variation between
the two, and Pinsker's inequality turns that into ε ≥ 2(AUC − ½)² nats. It is a **lower bound** only: a detector
at chance shows that this detector found no gap, not that ε is small. The bracket applies the same formula across
the file AUC's interval.

Chi-square, SPA and RS look for bits being overwritten, which Mist never does, so they are expected to sit at
chance; a rise means the embedder has drifted. HCF-COM looks for ±1 changes, which is exactly what Mist does,
so it is the classical detector that matters.

The **classifier** is the adversary of record. It is a logistic regression trained on this run's own clean and
stego chunks, using every detector's score, the share of values at each of -3…3, and how each step between
adjacent values follows the one before it. It is cross-validated by carrier: every carrier is scored by a model
trained without it, so it cannot win by memorising a track. Its message-size check trains a second model to
tell a 64-byte message from a 1-byte one.

**Markov** is the same kind of model, trained the same way, on one richer feature set only: how the second
difference between samples, the waveform's curvature, changes from one sample to the next, with each value
clipped to -3…3. That curvature is near zero wherever the audio is smooth, so ±1 changes stand out in it more
than in the values or their steps. These are the rich-model features of audio steganalysis; a detector that
learns its own features, a CNN trained on exported chunks, is the next step beyond them and is not run here.

**Message size** compares a 64-byte message with a 1-byte one. Mist changes the same amount of audio whatever
the message, so this should read 0.5: anything else means the message length shows.

### Audio quality

**SDR** (signal-to-distortion ratio) is how loud the music is compared with the error added to it, in dB. Higher
is better, and every 10 dB means ten times less error. Both copies are compared with the original carrier,
after lining them up in time. The **plain re-encode** is Mist's own re-encode with nothing embedded; the **embedding cost** is how
much lower the stego copy scores, which is Mist's own share. **Extra error energy** says the same thing as a
percentage: +37% means 37% more error than the plain re-encode alone.

The embedding cost is relative to the re-encode's own error, so the same perturbation reads as a larger cost on a
cleaner, higher-bitrate re-encode. **Mist's error below the music** does not depend on that: it compares the
stego copy with Mist's own re-encode, so the only difference left is what Mist changed, measured against the music
itself. The **per carrier** table lists every track with its output bitrate, so one odd track cannot hide in the
mean.

### Thresholds

| Measure | ✅ | ⚠️ | ❌ |
|---|---|---|---|
| Plain properties | every carrier matches ffmpeg | — | any carrier differs |
| Detector AUC | 95% interval includes 0.5 | interval excludes 0.5, AUC within 0.4–0.6 | AUC outside 0.4–0.6 |
| Message size | 0.45–0.55 | — | outside 0.45–0.55 |
| Embedding cost | ≤ 0.3 dB, or Mist output ≥ 70 dB SDR | 0.3–1 dB | over 1 dB |

### Reference points

- **0.3 dB embedding cost** and **AUC ≈ 0.5** are the roadmap's Phase 1 exit criteria.
- **0.18 dB** is what the design doc measured for Ogg Vorbis on one 128 kbps rock MP3.
- **70 dB SDR** puts the error far below anything audible at normal listening levels. 16-bit audio's own
  rounding noise sits about 96 dB down, which is why a lossless output can score ∞ for a plain re-encode
  (bit-exact) and pass on SDR alone.
- **A lossy carrier into a lossless output** is not bit-exact: an MP3 decodes to values between 16-bit steps,
  so the plain re-encode lands around 80–90 dB rather than ∞.
