# Mist harness report

 perceptual metric: not installed

Every carrier is encoded three times: by the ffmpeg command line at its own defaults (the *clean* copy, which is what a warden without the original would compare against), by Mist's own encoder with nothing embedded (Mist's *own* re-encode), and by Mist with a hidden message (the *stego* copy). The report asks whether the stego copy differs from the clean one in plain properties, whether a detector can tell them apart, and how much worse it sounds. [How to read this report](#how-to-read-this-report) explains every number and threshold.

## Summary

| Format | Carriers | Looks like ffmpeg? | Hidden from detectors? | Audio quality |
|---|---|---|---|---|
| flac | 84 / 84 | ❌ differs on 41 of 84 carriers: length, zero tail | ⚠️ faint signal from chi-square (AUC 0.501) | ✅ inaudible: the added error is at least 90.92 dB below the music |
| wav/pcm_s16le | 84 / 84 | ❌ differs on 41 of 84 carriers: length, zero tail | ⚠️ faint signal from chi-square (AUC 0.500) | ✅ inaudible: the added error is at least 80.80 dB below the music |
| ogg/vorbis | 84 / 84 | ❌ differs on 79 of 84 carriers: length, nominal bitrate, zero tail | ⚠️ faint signal from spa (AUC 0.499) | ✅ Mist's own error sits 51.63 dB below the music; embedding costs 0.02 dB, within the 0.3 dB target |

## flac

Measured on 84 of 84 carriers.

### Does it look like a plain ffmpeg encode?

The ffmpeg column is ffmpeg at its own defaults. Ogg Vorbis keeps the source's quality, so it differs from ffmpeg's default q3 whenever the carrier maps to another level.

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| 03 - Disposable Heroes.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| 04 - No Remorse.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| 06 - For Whom the Bell Tolls.mp3 | 11669760 | 11669231 / 11669760 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| 07 - The Four Horsemen.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| 08 - Fade to Black.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| 09 - Seek & Destroy.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| 10 - Whiplash.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| 11 - Fight Fire with Fire.mp3 | 11631744 | 11631215 / 11631744 | 288 / 288 | s32 / s32 | 0 / 0 | length |
| 14 - Motorbreath.mp3 | 12556800 | 12556271 / 12556800 | 30761 / 30761 | s32 / s32 | 0 / 0 | length |
| ATB.06.20.2025/01 ATB 06-20-2025.mp3 | 13231872 | 13230767 / 13231872 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| ATB.06.20.2025/02 ATB 06-20-2025.mp3 | 13231872 | 13230767 / 13231872 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| ATB.06.20.2025/03 ATB 06-20-2025.mp3 | 13231872 | 13230767 / 13231872 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| ATB.06.20.2025/04 ATB 06-20-2025.mp3 | 13231872 | 13230767 / 13231872 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| Pillars_of_the_Sky.wav | 2880000 | 2880000 / 2880000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| Starlight_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| Stellar_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side1.flac | 28803072 | 28803072 / 28803072 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side2.flac | 28803072 | 28803072 / 28803072 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t01.flac | 27650560 | 27650560 / 27650560 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t02.flac | 27998720 | 27998720 / 27998720 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t03.flac | 24209920 | 24209920 / 24209920 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t04.flac | 28803072 | 28803072 / 28803072 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t05.flac | 24574720 | 24574720 / 24574720 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t06.flac | 26830080 | 26830080 / 26830080 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t07.flac | 24695040 | 24695040 / 24695040 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t08.flac | 18909440 | 18909440 / 18909440 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t09.flac | 28803072 | 28803072 / 28803072 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/01 Track01.flac | 7939176 | 7939176 / 7939176 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/02 Track02.flac | 8334312 | 8334312 / 8334312 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/03 Track03.flac | 8187312 | 8187312 / 8187312 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/04 Track04.flac | 7616952 | 7616952 / 7616952 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/05 Track05.flac | 7703976 | 7703976 / 7703976 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/06 Track06.flac | 7795116 | 7795116 / 7795116 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/07 Track07.flac | 8786484 | 8786484 / 8786484 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/08 Track08.flac | 8306676 | 8306676 / 8306676 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/09 Track09.flac | 7848624 | 7848624 / 7848624 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/10 Track10.flac | 7849212 | 7849212 / 7849212 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/11 Track11.flac | 8371356 | 8371356 / 8371356 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/12 Track12.flac | 7808052 | 7808052 / 7808052 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/13 Track13.flac | 7792764 | 7792764 / 7792764 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/14 Track14.flac | 7705152 | 7705152 / 7705152 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/15 Track15.flac | 7733376 | 7733376 / 7733376 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/16 Track16.flac | 8332548 | 8332548 / 8332548 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/17 Track17.flac | 7836864 | 7836864 / 7836864 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/18 Track18.flac | 7783356 | 7783356 / 7783356 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/19 Track19.flac | 8134980 | 8134980 / 8134980 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/20 Track20.flac | 7904484 | 7904484 / 7904484 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/21 Track21.flac | 7577556 | 7577556 / 7577556 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/22 Track22.flac | 8433684 | 8433684 / 8433684 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/23 Track23.flac | 8167320 | 8167320 / 8167320 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/24 Track24.flac | 7443492 | 7443492 / 7443492 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/25 Track25.flac | 7847448 | 7847448 / 7847448 | 0 / 0 | s16 / s16 | 0 / 0 | — |
| sundaytimes-wind-in-the-willows/26 Track26.flac | 6560316 | 6560316 / 6560316 | 20 / 20 | s16 / s16 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Adam Ellis - Broken (& Jo Cartwright).opus | 7568328 | 7567536 / 7568328 | 1004 / 1796 | s32 / s32 | 0 / 0 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Hurt of intention 2019 (& Neev Kennedy).opus | 10717128 | 10716829 / 10717128 | 2375 / 2674 | s32 / s32 | 0 / 0 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Right back.opus | 10753608 | 10753254 / 10753608 | 462 / 816 | s32 / s32 | 0 / 0 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Alex Leavon - When I'm with you (& Julia Ross).opus | 8507208 | 8507078 / 8507208 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Ana Criado - In a thousand skies.opus | 9099528 | 9099131 / 9099528 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Blue5even - Through the barricades (& Jo Cartwright).opus | 10184328 | 10184328 / 10184328 | 0 / 0 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Braulio Stefield - See ghosts (& Victoriya).opus | 9849288 | 9848955 / 9849288 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Costa - Always (& Cathy Burton).opus | 10444488 | 10444219 / 10444488 | 1522 / 1791 | s32 / s32 | 0 / 0 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Delta-S - Letting go (Ikerya Project remix) (& Kate Louise Smith).opus | 10854408 | 10854001 / 10854408 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Derek Ryan - After dark (ft Melissa R. Kaplan).opus | 9693768 | 9693408 / 9693768 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Drival - Saviour (& Michele C).opus | 9767688 | 9766958 / 9767688 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/F.G. Noise - Waiting for the thunder (Killing time) (& Lauren Ní Chasaide).opus | 9360648 | 9359938 / 9360648 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Kaimo K - Hold of you (Denis Kenzo remix).opus | 9855048 | 9855000 / 9855048 | 8479 / 8527 | s32 / s32 | 0 / 0 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Kaimo K - When you come home (Myde rework).opus | 9146568 | 9146026 / 9146568 | 276 / 818 | s32 / s32 | 0 / 0 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Karanda - Still got time (& Sarah Russell).opus | 10718088 | 10717286 / 10718088 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Limelght - Run & hide (ft Alina Renae).opus | 8370888 | 8370000 / 8370888 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Lost Witness - Sewn.opus | 10456968 | 10456119 / 10456968 | 23958 / 24807 | s32 / s32 | 0 / 0 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Mhammed El Alami - Warriors (& Emma Horan).opus | 9574728 | 9573914 / 9574728 | 7702 / 8516 | s32 / s32 | 0 / 0 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Nicholas Gunn - Older (Costa remix) (ft Alina Renae).opus | 10171848 | 10171848 / 10171848 | 826 / 826 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Nitrous Oxide - Lower than the ground (& Sarah Russell).opus | 9736008 | 9735652 / 9736008 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Passenger 75 - Heartless (& Score).opus | 9412488 | 9412488 / 9412488 | 7538 / 7538 | s32 / s32 | 0 / 0 | — |
| va-female-vocal-trance-2019-opus-128/Perpetual - Innocent (ft Fisher).opus | 9888648 | 9888000 / 9888648 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Raz Nitzan - Beyond time (Aurosonic remix) (& Ellie Lawson).opus | 10440648 | 10440000 / 10440648 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Ronski Speed - Beat alive (Denis Airwave remix) (& Sarah Lynn).opus | 11520648 | 11520002 / 11520648 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Rub!k - Everglow (& Sue McLaren).opus | 10451208 | 10450747 / 10451208 | 21505 / 21966 | s32 / s32 | 0 / 0 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Stargazers - Be here with me (ft Katty Heath).opus | 8598408 | 8598259 / 8598408 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/Stargazers - Crystalize (& Fenna Day).opus | 11520648 | 11520000 / 11520648 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/The Blizzard - Always a stranger (Nitrous Oxide remix) (& Carol Lee).opus | 11103048 | 11102609 / 11103048 | 0 / 0 | s32 / s32 | 0 / 0 | length |
| va-female-vocal-trance-2019-opus-128/The City Never Sleeps - Addicted (NyTiGen remix) (& Summer Haze).opus | 8540808 | 8540309 / 8540808 | 7055 / 7554 | s32 / s32 | 0 / 0 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Whiteout - The part in-between (Wilderness & A-Line remix) (& One Half Bear).opus | 10280328 | 10279847 / 10280328 | 7063 / 7544 | s32 / s32 | 0 / 0 | length, zero tail |

**Verdict:** ❌ differs on 41 of 84 carriers: length, zero tail.

### Can a detector tell?

Stego copy against the clean ffmpeg copy (for Ogg Vorbis, ffmpeg at the quality level Mist chose, so only the embedding differs).

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.501 (0.500–0.501) | 0.508 (0.502–0.518) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.503 (0.498–0.507) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.501 (0.500–0.503) | 0.509 (0.500–0.518) | 0.000 (0.000–0.001) | ⚠️ faint signal per file | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.501 (0.499–0.504) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.514 (0.507–0.523) | 0.548 (0.523–0.577) | 0.005 (0.001–0.012) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.508 (0.503–0.513) | 0.544 (0.519–0.573) | 0.004 (0.001–0.011) | ⚠️ faint signal | ✅ hidden (0.500) |
| key-aware | the ephemeral key in the first frame's envelope, read with the public key alone | 0.512 (0.494–0.535) | 0.512 (0.494–0.535) | 0.000 (0.000–0.003) | ✅ chance | ✅ hidden (0.500) |

**Verdict:** ⚠️ faint signal from chi-square (AUC 0.501).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.501 (0.499–0.503) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.503 (0.499–0.507) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.506 (0.499–0.512) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.500 (0.499–0.501) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.514 (0.507–0.523) | 0.549 (0.524–0.577) | 0.005 (0.001–0.012) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.508 (0.503–0.513) | 0.546 (0.522–0.575) | 0.004 (0.001–0.011) | ⚠️ faint signal | ✅ hidden (0.500) |

### Does detection grow with audio?

File AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.

| Detector | First 40 chunks | First 240 chunks | Whole file |
|---|---|---|---|
| chi-square | 0.522 (0.510–0.540) | 0.512 (0.504–0.526) | 0.508 (0.502–0.518) |
| spa | 0.498 (0.492–0.504) | 0.502 (0.498–0.507) | 0.503 (0.498–0.507) |
| rs | 0.503 (0.492–0.513) | 0.509 (0.502–0.518) | 0.509 (0.500–0.518) |
| hcf-com | 0.500 (0.499–0.500) | 0.500 (0.499–0.502) | 0.501 (0.499–0.504) |
| classifier | 0.530 (0.508–0.555) | 0.539 (0.514–0.567) | 0.548 (0.523–0.577) |
| markov | 0.529 (0.509–0.554) | 0.525 (0.507–0.547) | 0.544 (0.519–0.573) |

### By kind of audio

File AUC (95%) against the clean ffmpeg copy, per corpus folder. The count is carriers; few carriers means a wide interval.

| Detector | (top level) (13) | ATB.06.20.2025 (4) | disc1 (2) | hmbb2015-03-18.sbd.flac24 (9) | sundaytimes-wind-in-the-willows (26) | va-female-vocal-trance-2019-opus-128 (30) |
|---|---|---|---|---|---|---|
| chi-square | 0.680 (0.536–0.913) | 0.562 (0.417–0.667) | 0.375 (0.000–0.500) | 0.512 (0.487–0.538) | 0.499 (0.483–0.512) | 0.519 (0.501–0.544) |
| spa | 0.479 (0.403–0.546) | 0.562 (0.417–0.667) | 0.500 (0.000–1.000) | 0.531 (0.494–0.561) | 0.515 (0.486–0.545) | 0.506 (0.492–0.521) |
| rs | 0.536 (0.425–0.685) | 0.500 (0.333–0.667) | 0.500 (0.000–1.000) | 0.568 (0.481–0.689) | 0.524 (0.486–0.567) | 0.524 (0.488–0.561) |
| hcf-com | 0.500 (0.488–0.512) | 0.531 (0.500–0.625) | 0.500 (0.000–1.000) | 0.543 (0.519–0.561) | 0.494 (0.488–0.501) | 0.508 (0.496–0.523) |
| classifier | 0.639 (0.534–0.792) | 0.938 (0.667–1.000) | 0.250 (0.000–0.333) | 1.000 (1.000–1.000) | 0.503 (0.496–0.510) | 0.978 (0.944–0.998) |
| markov | 0.704 (0.590–0.865) | 0.938 (0.667–1.000) | 0.250 (0.000–0.333) | 0.556 (0.551–0.565) | 0.496 (0.488–0.503) | 0.927 (0.861–0.977) |

### Does detection grow with the number of files?

AUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.

| Detector | 1 file(s) | 3 file(s) | 6 file(s) |
|---|---|---|---|
| chi-square | 0.494 | 0.484 | 0.485 |
| spa | 0.531 | 0.505 | 0.516 |
| rs | 0.538 | 0.510 | 0.517 |
| hcf-com | 0.491 | 0.484 | 0.482 |
| classifier | 0.561 | 0.555 | 0.550 |
| markov | 0.554 | 0.548 | 0.543 |

### A learned warden

A small CNN trained on the first 16 chunks of each carrier, five folds split by carrier, one score per file (`tools/cnn_warden`).

| Detector | Carriers | File AUC (95%) | ε ≥ (nats, 95%) |
|---|---|---|---|
| cnn | 84 | 0.501 (0.496–0.508) | 0.000 (0.000–0.000) |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 65.46 dB | 35.91 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 85.24 dB | 35.91 dB | The same, with the message embedded |
| Embedding cost | 0.01 dB | 0.04 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +1% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 131.13 dB | 90.92 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ inaudible: the added error is at least 90.92 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 1594 | 128.79 dB | 128.75 dB | 0.04 dB | 149.02 dB |
| 03 - Disposable Heroes.mp3 | 1588 | 128.91 dB | 128.86 dB | 0.04 dB | 149.13 dB |
| 04 - No Remorse.mp3 | 1575 | 130.61 dB | 130.57 dB | 0.04 dB | 150.71 dB |
| 06 - For Whom the Bell Tolls.mp3 | 1545 | 128.38 dB | 128.34 dB | 0.04 dB | 148.64 dB |
| 07 - The Four Horsemen.mp3 | 1563 | 129.30 dB | 129.25 dB | 0.04 dB | 149.49 dB |
| 08 - Fade to Black.mp3 | 1574 | 129.70 dB | 129.66 dB | 0.04 dB | 149.89 dB |
| 09 - Seek & Destroy.mp3 | 1598 | 130.09 dB | 130.05 dB | 0.04 dB | 150.24 dB |
| 10 - Whiplash.mp3 | 1550 | 130.75 dB | 130.70 dB | 0.04 dB | 150.86 dB |
| 11 - Fight Fire with Fire.mp3 | 1531 | 128.87 dB | 128.83 dB | 0.04 dB | 149.14 dB |
| 14 - Motorbreath.mp3 | 1449 | 129.02 dB | 128.98 dB | 0.04 dB | 149.29 dB |
| ATB.06.20.2025/01 ATB 06-20-2025.mp3 | 1518 | 64.22 dB | 64.22 dB | 0.00 dB | 152.10 dB |
| ATB.06.20.2025/02 ATB 06-20-2025.mp3 | 1663 | 50.07 dB | 50.07 dB | 0.00 dB | 151.19 dB |
| ATB.06.20.2025/03 ATB 06-20-2025.mp3 | 1553 | 46.00 dB | 46.00 dB | 0.00 dB | 149.43 dB |
| ATB.06.20.2025/04 ATB 06-20-2025.mp3 | 1443 | 43.92 dB | 43.92 dB | 0.00 dB | 149.98 dB |
| Pillars_of_the_Sky.wav | 968 | — | 99.92 dB | — | 99.92 dB |
| Starlight_Ascent.wav | 1046 | — | 102.12 dB | — | 102.12 dB |
| Stellar_Ascent.wav | 978 | — | 102.86 dB | — | 102.86 dB |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side1.flac | 2297 | — | 131.50 dB | — | 131.50 dB |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side2.flac | 2258 | — | 128.95 dB | — | 128.95 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t01.flac | 2593 | — | 146.93 dB | — | 146.93 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t02.flac | 2648 | — | 146.66 dB | — | 146.66 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t03.flac | 2676 | — | 148.17 dB | — | 148.17 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t04.flac | 2583 | — | 145.48 dB | — | 145.48 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t05.flac | 2685 | — | 146.50 dB | — | 146.50 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t06.flac | 2698 | — | 146.86 dB | — | 146.86 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t07.flac | 2688 | — | 147.34 dB | — | 147.34 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t08.flac | 2661 | — | 147.32 dB | — | 147.32 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t09.flac | 2638 | — | 146.83 dB | — | 146.83 dB |
| sundaytimes-wind-in-the-willows/01 Track01.flac | 467 | — | 92.43 dB | — | 92.43 dB |
| sundaytimes-wind-in-the-willows/02 Track02.flac | 464 | — | 92.13 dB | — | 92.13 dB |
| sundaytimes-wind-in-the-willows/03 Track03.flac | 461 | — | 92.03 dB | — | 92.03 dB |
| sundaytimes-wind-in-the-willows/04 Track04.flac | 467 | — | 91.95 dB | — | 91.95 dB |
| sundaytimes-wind-in-the-willows/05 Track05.flac | 467 | — | 92.45 dB | — | 92.45 dB |
| sundaytimes-wind-in-the-willows/06 Track06.flac | 458 | — | 91.77 dB | — | 91.77 dB |
| sundaytimes-wind-in-the-willows/07 Track07.flac | 465 | — | 91.96 dB | — | 91.96 dB |
| sundaytimes-wind-in-the-willows/08 Track08.flac | 470 | — | 92.36 dB | — | 92.36 dB |
| sundaytimes-wind-in-the-willows/09 Track09.flac | 467 | — | 91.74 dB | — | 91.74 dB |
| sundaytimes-wind-in-the-willows/10 Track10.flac | 465 | — | 91.96 dB | — | 91.96 dB |
| sundaytimes-wind-in-the-willows/11 Track11.flac | 463 | — | 91.75 dB | — | 91.75 dB |
| sundaytimes-wind-in-the-willows/12 Track12.flac | 466 | — | 91.75 dB | — | 91.75 dB |
| sundaytimes-wind-in-the-willows/13 Track13.flac | 466 | — | 92.12 dB | — | 92.12 dB |
| sundaytimes-wind-in-the-willows/14 Track14.flac | 474 | — | 92.17 dB | — | 92.17 dB |
| sundaytimes-wind-in-the-willows/15 Track15.flac | 460 | — | 90.92 dB | — | 90.92 dB |
| sundaytimes-wind-in-the-willows/16 Track16.flac | 468 | — | 91.80 dB | — | 91.80 dB |
| sundaytimes-wind-in-the-willows/17 Track17.flac | 483 | — | 92.86 dB | — | 92.86 dB |
| sundaytimes-wind-in-the-willows/18 Track18.flac | 459 | — | 92.22 dB | — | 92.22 dB |
| sundaytimes-wind-in-the-willows/19 Track19.flac | 470 | — | 92.26 dB | — | 92.26 dB |
| sundaytimes-wind-in-the-willows/20 Track20.flac | 468 | — | 92.39 dB | — | 92.39 dB |
| sundaytimes-wind-in-the-willows/21 Track21.flac | 462 | — | 92.02 dB | — | 92.02 dB |
| sundaytimes-wind-in-the-willows/22 Track22.flac | 477 | — | 92.11 dB | — | 92.11 dB |
| sundaytimes-wind-in-the-willows/23 Track23.flac | 462 | — | 92.03 dB | — | 92.03 dB |
| sundaytimes-wind-in-the-willows/24 Track24.flac | 463 | — | 91.94 dB | — | 91.94 dB |
| sundaytimes-wind-in-the-willows/25 Track25.flac | 465 | — | 91.84 dB | — | 91.84 dB |
| sundaytimes-wind-in-the-willows/26 Track26.flac | 473 | — | 91.83 dB | — | 91.83 dB |
| va-female-vocal-trance-2019-opus-128/Adam Ellis - Broken (& Jo Cartwright).opus | 1874 | 46.77 dB | 46.77 dB | 0.00 dB | 154.53 dB |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Hurt of intention 2019 (& Neev Kennedy).opus | 1947 | 36.65 dB | 36.65 dB | 0.00 dB | 156.43 dB |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Right back.opus | 1832 | 45.62 dB | 45.62 dB | 0.00 dB | 153.94 dB |
| va-female-vocal-trance-2019-opus-128/Alex Leavon - When I'm with you (& Julia Ross).opus | 1910 | 35.91 dB | 35.91 dB | 0.00 dB | 154.82 dB |
| va-female-vocal-trance-2019-opus-128/Ana Criado - In a thousand skies.opus | 1936 | 44.31 dB | 44.31 dB | 0.00 dB | 154.73 dB |
| va-female-vocal-trance-2019-opus-128/Blue5even - Through the barricades (& Jo Cartwright).opus | 1873 | 58.54 dB | 58.54 dB | 0.00 dB | 151.85 dB |
| va-female-vocal-trance-2019-opus-128/Braulio Stefield - See ghosts (& Victoriya).opus | 1854 | 41.20 dB | 41.20 dB | 0.00 dB | 154.85 dB |
| va-female-vocal-trance-2019-opus-128/Costa - Always (& Cathy Burton).opus | 1916 | 41.33 dB | 41.33 dB | 0.00 dB | 155.28 dB |
| va-female-vocal-trance-2019-opus-128/Delta-S - Letting go (Ikerya Project remix) (& Kate Louise Smith).opus | 1929 | 43.97 dB | 43.97 dB | 0.00 dB | 155.78 dB |
| va-female-vocal-trance-2019-opus-128/Derek Ryan - After dark (ft Melissa R. Kaplan).opus | 1769 | 68.84 dB | 68.84 dB | 0.00 dB | 151.70 dB |
| va-female-vocal-trance-2019-opus-128/Drival - Saviour (& Michele C).opus | 1860 | 52.48 dB | 52.48 dB | 0.00 dB | 153.86 dB |
| va-female-vocal-trance-2019-opus-128/F.G. Noise - Waiting for the thunder (Killing time) (& Lauren Ní Chasaide).opus | 1894 | 43.08 dB | 43.08 dB | 0.00 dB | 154.58 dB |
| va-female-vocal-trance-2019-opus-128/Kaimo K - Hold of you (Denis Kenzo remix).opus | 1965 | 46.30 dB | 46.30 dB | 0.00 dB | 155.47 dB |
| va-female-vocal-trance-2019-opus-128/Kaimo K - When you come home (Myde rework).opus | 1927 | 38.87 dB | 38.87 dB | 0.00 dB | 156.25 dB |
| va-female-vocal-trance-2019-opus-128/Karanda - Still got time (& Sarah Russell).opus | 1850 | 54.46 dB | 54.46 dB | 0.00 dB | 154.48 dB |
| va-female-vocal-trance-2019-opus-128/Limelght - Run & hide (ft Alina Renae).opus | 1767 | 47.32 dB | 47.32 dB | 0.00 dB | 153.67 dB |
| va-female-vocal-trance-2019-opus-128/Lost Witness - Sewn.opus | 1891 | 43.18 dB | 43.18 dB | 0.00 dB | 153.78 dB |
| va-female-vocal-trance-2019-opus-128/Mhammed El Alami - Warriors (& Emma Horan).opus | 1921 | 45.06 dB | 45.06 dB | 0.00 dB | 155.45 dB |
| va-female-vocal-trance-2019-opus-128/Nicholas Gunn - Older (Costa remix) (ft Alina Renae).opus | 1851 | 49.41 dB | 49.41 dB | 0.00 dB | 152.73 dB |
| va-female-vocal-trance-2019-opus-128/Nitrous Oxide - Lower than the ground (& Sarah Russell).opus | 1899 | 40.06 dB | 40.06 dB | 0.00 dB | 155.19 dB |
| va-female-vocal-trance-2019-opus-128/Passenger 75 - Heartless (& Score).opus | 1775 | 44.09 dB | 44.09 dB | 0.00 dB | 154.61 dB |
| va-female-vocal-trance-2019-opus-128/Perpetual - Innocent (ft Fisher).opus | 1984 | 39.55 dB | 39.55 dB | 0.00 dB | 155.76 dB |
| va-female-vocal-trance-2019-opus-128/Raz Nitzan - Beyond time (Aurosonic remix) (& Ellie Lawson).opus | 1917 | 48.44 dB | 48.44 dB | 0.00 dB | 155.15 dB |
| va-female-vocal-trance-2019-opus-128/Ronski Speed - Beat alive (Denis Airwave remix) (& Sarah Lynn).opus | 1914 | 40.85 dB | 40.85 dB | 0.00 dB | 156.08 dB |
| va-female-vocal-trance-2019-opus-128/Rub!k - Everglow (& Sue McLaren).opus | 1932 | 45.88 dB | 45.88 dB | 0.00 dB | 154.63 dB |
| va-female-vocal-trance-2019-opus-128/Stargazers - Be here with me (ft Katty Heath).opus | 1938 | 46.94 dB | 46.94 dB | 0.00 dB | 154.75 dB |
| va-female-vocal-trance-2019-opus-128/Stargazers - Crystalize (& Fenna Day).opus | 1898 | 48.47 dB | 48.47 dB | 0.00 dB | 154.33 dB |
| va-female-vocal-trance-2019-opus-128/The Blizzard - Always a stranger (Nitrous Oxide remix) (& Carol Lee).opus | 1851 | 43.09 dB | 43.09 dB | 0.00 dB | 153.62 dB |
| va-female-vocal-trance-2019-opus-128/The City Never Sleeps - Addicted (NyTiGen remix) (& Summer Haze).opus | 1902 | 59.77 dB | 59.77 dB | 0.00 dB | 151.95 dB |
| va-female-vocal-trance-2019-opus-128/Whiteout - The part in-between (Wilderness & A-Line remix) (& One Half Bear).opus | 1813 | 41.02 dB | 41.02 dB | 0.00 dB | 155.39 dB |

## wav/pcm_s16le

Measured on 84 of 84 carriers.

### Does it look like a plain ffmpeg encode?

The ffmpeg column is ffmpeg at its own defaults. Ogg Vorbis keeps the source's quality, so it differs from ffmpeg's default q3 whenever the carrier maps to another level.

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s16 / s16 | 1411 / 1411 | length |
| 03 - Disposable Heroes.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s16 / s16 | 1411 / 1411 | length |
| 04 - No Remorse.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s16 / s16 | 1411 / 1411 | length |
| 06 - For Whom the Bell Tolls.mp3 | 11669760 | 11669231 / 11669760 | 0 / 0 | s16 / s16 | 1411 / 1411 | length |
| 07 - The Four Horsemen.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s16 / s16 | 1411 / 1411 | length |
| 08 - Fade to Black.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s16 / s16 | 1411 / 1411 | length |
| 09 - Seek & Destroy.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s16 / s16 | 1411 / 1411 | length |
| 10 - Whiplash.mp3 | 13230720 | 13230191 / 13230720 | 0 / 0 | s16 / s16 | 1411 / 1411 | length |
| 11 - Fight Fire with Fire.mp3 | 11631744 | 11631215 / 11631744 | 300 / 300 | s16 / s16 | 1411 / 1411 | length |
| 14 - Motorbreath.mp3 | 12556800 | 12556271 / 12556800 | 30781 / 30781 | s16 / s16 | 1411 / 1411 | length |
| ATB.06.20.2025/01 ATB 06-20-2025.mp3 | 13231872 | 13230767 / 13231872 | 0 / 0 | s16 / s16 | 1411 / 1411 | length |
| ATB.06.20.2025/02 ATB 06-20-2025.mp3 | 13231872 | 13230767 / 13231872 | 0 / 0 | s16 / s16 | 1411 / 1411 | length |
| ATB.06.20.2025/03 ATB 06-20-2025.mp3 | 13231872 | 13230767 / 13231872 | 0 / 0 | s16 / s16 | 1411 / 1411 | length |
| ATB.06.20.2025/04 ATB 06-20-2025.mp3 | 13231872 | 13230767 / 13231872 | 0 / 0 | s16 / s16 | 1411 / 1411 | length |
| Pillars_of_the_Sky.wav | 2880000 | 2880000 / 2880000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| Starlight_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| Stellar_Ascent.wav | 5760000 | 5760000 / 5760000 | 0 / 0 | s16 / s16 | 1536 / 1536 | — |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side1.flac | 28803072 | 28803072 / 28803072 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side2.flac | 28803072 | 28803072 / 28803072 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t01.flac | 27650560 | 27650560 / 27650560 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t02.flac | 27998720 | 27998720 / 27998720 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t03.flac | 24209920 | 24209920 / 24209920 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t04.flac | 28803072 | 28803072 / 28803072 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t05.flac | 24574720 | 24574720 / 24574720 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t06.flac | 26830080 | 26830080 / 26830080 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t07.flac | 24695040 | 24695040 / 24695040 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t08.flac | 18909440 | 18909440 / 18909440 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t09.flac | 28803072 | 28803072 / 28803072 | 0 / 0 | s16 / s16 | 3072 / 3072 | — |
| sundaytimes-wind-in-the-willows/01 Track01.flac | 7939176 | 7939176 / 7939176 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/02 Track02.flac | 8334312 | 8334312 / 8334312 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/03 Track03.flac | 8187312 | 8187312 / 8187312 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/04 Track04.flac | 7616952 | 7616952 / 7616952 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/05 Track05.flac | 7703976 | 7703976 / 7703976 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/06 Track06.flac | 7795116 | 7795116 / 7795116 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/07 Track07.flac | 8786484 | 8786484 / 8786484 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/08 Track08.flac | 8306676 | 8306676 / 8306676 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/09 Track09.flac | 7848624 | 7848624 / 7848624 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/10 Track10.flac | 7849212 | 7849212 / 7849212 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/11 Track11.flac | 8371356 | 8371356 / 8371356 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/12 Track12.flac | 7808052 | 7808052 / 7808052 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/13 Track13.flac | 7792764 | 7792764 / 7792764 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/14 Track14.flac | 7705152 | 7705152 / 7705152 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/15 Track15.flac | 7733376 | 7733376 / 7733376 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/16 Track16.flac | 8332548 | 8332548 / 8332548 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/17 Track17.flac | 7836864 | 7836864 / 7836864 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/18 Track18.flac | 7783356 | 7783356 / 7783356 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/19 Track19.flac | 8134980 | 8134980 / 8134980 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/20 Track20.flac | 7904484 | 7904484 / 7904484 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/21 Track21.flac | 7577556 | 7577556 / 7577556 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/22 Track22.flac | 8433684 | 8433684 / 8433684 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/23 Track23.flac | 8167320 | 8167320 / 8167320 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/24 Track24.flac | 7443492 | 7443492 / 7443492 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/25 Track25.flac | 7847448 | 7847448 / 7847448 | 0 / 0 | s16 / s16 | 1411 / 1411 | — |
| sundaytimes-wind-in-the-willows/26 Track26.flac | 6560316 | 6560316 / 6560316 | 20 / 20 | s16 / s16 | 1411 / 1411 | — |
| va-female-vocal-trance-2019-opus-128/Adam Ellis - Broken (& Jo Cartwright).opus | 7568328 | 7567536 / 7568328 | 2973 / 3765 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Hurt of intention 2019 (& Neev Kennedy).opus | 10717128 | 10716829 / 10717128 | 2698 / 2997 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Right back.opus | 10753608 | 10753254 / 10753608 | 466 / 820 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Alex Leavon - When I'm with you (& Julia Ross).opus | 8507208 | 8507078 / 8507208 | 3 / 8 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Ana Criado - In a thousand skies.opus | 9099528 | 9099131 / 9099528 | 0 / 8 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Blue5even - Through the barricades (& Jo Cartwright).opus | 10184328 | 10184328 / 10184328 | 1 / 1 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Braulio Stefield - See ghosts (& Victoriya).opus | 9849288 | 9848955 / 9849288 | 0 / 0 | s16 / s16 | 1536 / 1536 | length |
| va-female-vocal-trance-2019-opus-128/Costa - Always (& Cathy Burton).opus | 10444488 | 10444219 / 10444488 | 1526 / 1795 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Delta-S - Letting go (Ikerya Project remix) (& Kate Louise Smith).opus | 10854408 | 10854001 / 10854408 | 0 / 16 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Derek Ryan - After dark (ft Melissa R. Kaplan).opus | 9693768 | 9693408 / 9693768 | 1 / 8 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Drival - Saviour (& Michele C).opus | 9767688 | 9766958 / 9767688 | 1 / 8 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/F.G. Noise - Waiting for the thunder (Killing time) (& Lauren Ní Chasaide).opus | 9360648 | 9359938 / 9360648 | 1 / 9 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Kaimo K - Hold of you (Denis Kenzo remix).opus | 9855048 | 9855000 / 9855048 | 8479 / 8527 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Kaimo K - When you come home (Myde rework).opus | 9146568 | 9146026 / 9146568 | 280 / 822 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Karanda - Still got time (& Sarah Russell).opus | 10718088 | 10717286 / 10718088 | 3 / 8 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Limelght - Run & hide (ft Alina Renae).opus | 8370888 | 8370000 / 8370888 | 1 / 9 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Lost Witness - Sewn.opus | 10456968 | 10456119 / 10456968 | 23962 / 24811 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Mhammed El Alami - Warriors (& Emma Horan).opus | 9574728 | 9573914 / 9574728 | 8695 / 9509 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Nicholas Gunn - Older (Costa remix) (ft Alina Renae).opus | 10171848 | 10171848 / 10171848 | 830 / 830 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Nitrous Oxide - Lower than the ground (& Sarah Russell).opus | 9736008 | 9735652 / 9736008 | 0 / 8 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Passenger 75 - Heartless (& Score).opus | 9412488 | 9412488 / 9412488 | 7542 / 7542 | s16 / s16 | 1536 / 1536 | — |
| va-female-vocal-trance-2019-opus-128/Perpetual - Innocent (ft Fisher).opus | 9888648 | 9888000 / 9888648 | 0 / 8 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Raz Nitzan - Beyond time (Aurosonic remix) (& Ellie Lawson).opus | 10440648 | 10440000 / 10440648 | 2 / 9 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Ronski Speed - Beat alive (Denis Airwave remix) (& Sarah Lynn).opus | 11520648 | 11520002 / 11520648 | 0 / 9 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Rub!k - Everglow (& Sue McLaren).opus | 10451208 | 10450747 / 10451208 | 21515 / 21976 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Stargazers - Be here with me (ft Katty Heath).opus | 8598408 | 8598259 / 8598408 | 5 / 9 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Stargazers - Crystalize (& Fenna Day).opus | 11520648 | 11520000 / 11520648 | 0 / 8 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/The Blizzard - Always a stranger (Nitrous Oxide remix) (& Carol Lee).opus | 11103048 | 11102609 / 11103048 | 2 / 8 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/The City Never Sleeps - Addicted (NyTiGen remix) (& Summer Haze).opus | 8540808 | 8540309 / 8540808 | 7390 / 7889 | s16 / s16 | 1536 / 1536 | length, zero tail |
| va-female-vocal-trance-2019-opus-128/Whiteout - The part in-between (Wilderness & A-Line remix) (& One Half Bear).opus | 10280328 | 10279847 / 10280328 | 7068 / 7549 | s16 / s16 | 1536 / 1536 | length, zero tail |

**Verdict:** ❌ differs on 41 of 84 carriers: length, zero tail.

### Can a detector tell?

Stego copy against the clean ffmpeg copy (for Ogg Vorbis, ffmpeg at the quality level Mist chose, so only the embedding differs).

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.497 (0.490–0.503) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.499 (0.498–0.500) | 0.496 (0.491–0.500) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.497–0.503) | 0.501 (0.491–0.511) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.499) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.501 (0.499–0.503) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.506 (0.503–0.510) | 0.512 (0.504–0.522) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.503 (0.501–0.505) | 0.504 (0.501–0.507) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| key-aware | the ephemeral key in the first frame's envelope, read with the public key alone | 0.506 (0.482–0.530) | 0.506 (0.482–0.530) | 0.000 (0.000–0.002) | ✅ chance | ✅ hidden (0.500) |

**Verdict:** ⚠️ faint signal from chi-square (AUC 0.500).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.501) | 0.501 (0.497–0.503) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.497 (0.493–0.500) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.501) | 0.502 (0.496–0.509) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.499) |
| hcf-com | ±1 changes, which is what Mist does | 0.500 (0.500–0.500) | 0.501 (0.500–0.502) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.505 (0.502–0.509) | 0.510 (0.504–0.519) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.502 (0.501–0.504) | 0.504 (0.501–0.507) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |

### Does detection grow with audio?

File AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.

| Detector | First 40 chunks | First 240 chunks | Whole file |
|---|---|---|---|
| chi-square | 0.500 (0.492–0.506) | 0.498 (0.492–0.502) | 0.497 (0.490–0.503) |
| spa | 0.494 (0.488–0.499) | 0.498 (0.492–0.503) | 0.496 (0.491–0.500) |
| rs | 0.488 (0.471–0.503) | 0.499 (0.489–0.509) | 0.501 (0.491–0.511) |
| hcf-com | 0.507 (0.502–0.514) | 0.503 (0.500–0.507) | 0.501 (0.499–0.503) |
| classifier | 0.511 (0.504–0.520) | 0.520 (0.509–0.533) | 0.512 (0.504–0.522) |
| markov | 0.510 (0.502–0.522) | 0.512 (0.503–0.524) | 0.504 (0.501–0.507) |

### By kind of audio

File AUC (95%) against the clean ffmpeg copy, per corpus folder. The count is carriers; few carriers means a wide interval.

| Detector | (top level) (13) | ATB.06.20.2025 (4) | disc1 (2) | hmbb2015-03-18.sbd.flac24 (9) | sundaytimes-wind-in-the-willows (26) | va-female-vocal-trance-2019-opus-128 (30) |
|---|---|---|---|---|---|---|
| chi-square | 0.488 (0.421–0.523) | 0.625 (0.417–1.000) | 0.500 (0.000–1.000) | 0.488 (0.404–0.534) | 0.499 (0.491–0.506) | 0.510 (0.504–0.515) |
| spa | 0.473 (0.411–0.546) | 0.500 (0.333–0.667) | 0.500 (0.000–1.000) | 0.494 (0.456–0.533) | 0.496 (0.482–0.511) | 0.490 (0.470–0.503) |
| rs | 0.556 (0.376–0.771) | 0.312 (0.000–0.583) | 0.500 (0.000–1.000) | 0.457 (0.383–0.507) | 0.524 (0.493–0.560) | 0.501 (0.467–0.531) |
| hcf-com | 0.544 (0.521–0.578) | 0.688 (0.615–1.000) | 0.500 (0.000–1.000) | 0.506 (0.466–0.544) | 0.497 (0.490–0.504) | 0.498 (0.492–0.503) |
| classifier | 0.538 (0.497–0.609) | 0.438 (0.333–0.583) | 0.250 (0.000–0.333) | 0.531 (0.494–0.559) | 0.494 (0.485–0.503) | 0.784 (0.691–0.868) |
| markov | 0.556 (0.515–0.619) | 0.375 (0.333–0.400) | 0.250 (0.000–0.333) | 0.506 (0.468–0.546) | 0.506 (0.499–0.513) | 0.744 (0.647–0.836) |

### Does detection grow with the number of files?

AUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.

| Detector | 1 file(s) | 3 file(s) | 6 file(s) |
|---|---|---|---|
| chi-square | 0.540 | 0.491 | 0.508 |
| spa | 0.484 | 0.507 | 0.494 |
| rs | 0.488 | 0.509 | 0.496 |
| hcf-com | 0.474 | 0.482 | 0.476 |
| classifier | 0.537 | 0.527 | 0.517 |
| markov | 0.529 | 0.520 | 0.507 |

### A learned warden

A small CNN trained on the first 16 chunks of each carrier, five folds split by carrier, one score per file (`tools/cnn_warden`).

| Detector | Carriers | File AUC (95%) | ε ≥ (nats, 95%) |
|---|---|---|---|
| cnn | 84 | 0.503 (0.496–0.510) | 0.000 (0.000–0.000) |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 58.45 dB | 35.91 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 70.36 dB | 35.91 dB | The same, with the message embedded |
| Embedding cost | 0.01 dB | 0.04 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +1% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 99.57 dB | 80.80 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ inaudible: the added error is at least 80.80 dB below the music.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 1411 | 79.97 dB | 79.93 dB | 0.04 dB | 100.83 dB |
| 03 - Disposable Heroes.mp3 | 1411 | 80.09 dB | 80.05 dB | 0.04 dB | 100.94 dB |
| 04 - No Remorse.mp3 | 1411 | 81.66 dB | 81.63 dB | 0.04 dB | 102.52 dB |
| 06 - For Whom the Bell Tolls.mp3 | 1411 | 79.60 dB | 79.56 dB | 0.04 dB | 100.46 dB |
| 07 - The Four Horsemen.mp3 | 1411 | 80.45 dB | 80.42 dB | 0.04 dB | 101.31 dB |
| 08 - Fade to Black.mp3 | 1411 | 80.85 dB | 80.82 dB | 0.04 dB | 101.70 dB |
| 09 - Seek & Destroy.mp3 | 1411 | 81.19 dB | 81.16 dB | 0.04 dB | 102.05 dB |
| 10 - Whiplash.mp3 | 1411 | 81.82 dB | 81.78 dB | 0.04 dB | 102.67 dB |
| 11 - Fight Fire with Fire.mp3 | 1411 | 80.09 dB | 80.05 dB | 0.04 dB | 100.94 dB |
| 14 - Motorbreath.mp3 | 1411 | 80.24 dB | 80.20 dB | 0.04 dB | 101.10 dB |
| ATB.06.20.2025/01 ATB 06-20-2025.mp3 | 1411 | 64.16 dB | 64.16 dB | 0.00 dB | 103.91 dB |
| ATB.06.20.2025/02 ATB 06-20-2025.mp3 | 1411 | 50.06 dB | 50.06 dB | 0.00 dB | 103.00 dB |
| ATB.06.20.2025/03 ATB 06-20-2025.mp3 | 1411 | 45.99 dB | 45.99 dB | 0.00 dB | 101.24 dB |
| ATB.06.20.2025/04 ATB 06-20-2025.mp3 | 1411 | 43.92 dB | 43.92 dB | 0.00 dB | 101.79 dB |
| Pillars_of_the_Sky.wav | 1536 | — | 99.92 dB | — | 99.92 dB |
| Starlight_Ascent.wav | 1536 | — | 102.10 dB | — | 102.10 dB |
| Stellar_Ascent.wav | 1536 | — | 102.82 dB | — | 102.82 dB |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side1.flac | 3072 | 62.48 dB | 62.45 dB | 0.04 dB | 83.35 dB |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side2.flac | 3072 | 59.93 dB | 59.89 dB | 0.04 dB | 80.80 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t01.flac | 3072 | 77.91 dB | 77.87 dB | 0.04 dB | 98.74 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t02.flac | 3072 | 77.64 dB | 77.60 dB | 0.04 dB | 98.47 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t03.flac | 3072 | 79.15 dB | 79.11 dB | 0.04 dB | 99.98 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t04.flac | 3072 | 76.46 dB | 76.42 dB | 0.04 dB | 97.29 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t05.flac | 3072 | 77.47 dB | 77.43 dB | 0.04 dB | 98.30 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t06.flac | 3072 | 77.83 dB | 77.80 dB | 0.04 dB | 98.67 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t07.flac | 3072 | 78.32 dB | 78.28 dB | 0.04 dB | 99.15 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t08.flac | 3072 | 78.30 dB | 78.26 dB | 0.04 dB | 99.13 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t09.flac | 3072 | 77.81 dB | 77.77 dB | 0.04 dB | 98.64 dB |
| sundaytimes-wind-in-the-willows/01 Track01.flac | 1411 | — | 92.39 dB | — | 92.39 dB |
| sundaytimes-wind-in-the-willows/02 Track02.flac | 1411 | — | 92.11 dB | — | 92.11 dB |
| sundaytimes-wind-in-the-willows/03 Track03.flac | 1411 | — | 92.00 dB | — | 92.00 dB |
| sundaytimes-wind-in-the-willows/04 Track04.flac | 1411 | — | 91.92 dB | — | 91.92 dB |
| sundaytimes-wind-in-the-willows/05 Track05.flac | 1411 | — | 92.42 dB | — | 92.42 dB |
| sundaytimes-wind-in-the-willows/06 Track06.flac | 1411 | — | 91.73 dB | — | 91.73 dB |
| sundaytimes-wind-in-the-willows/07 Track07.flac | 1411 | — | 91.93 dB | — | 91.93 dB |
| sundaytimes-wind-in-the-willows/08 Track08.flac | 1411 | — | 92.33 dB | — | 92.33 dB |
| sundaytimes-wind-in-the-willows/09 Track09.flac | 1411 | — | 91.70 dB | — | 91.70 dB |
| sundaytimes-wind-in-the-willows/10 Track10.flac | 1411 | — | 91.92 dB | — | 91.92 dB |
| sundaytimes-wind-in-the-willows/11 Track11.flac | 1411 | — | 91.72 dB | — | 91.72 dB |
| sundaytimes-wind-in-the-willows/12 Track12.flac | 1411 | — | 91.71 dB | — | 91.71 dB |
| sundaytimes-wind-in-the-willows/13 Track13.flac | 1411 | — | 92.08 dB | — | 92.08 dB |
| sundaytimes-wind-in-the-willows/14 Track14.flac | 1411 | — | 92.14 dB | — | 92.14 dB |
| sundaytimes-wind-in-the-willows/15 Track15.flac | 1411 | — | 90.88 dB | — | 90.88 dB |
| sundaytimes-wind-in-the-willows/16 Track16.flac | 1411 | — | 91.77 dB | — | 91.77 dB |
| sundaytimes-wind-in-the-willows/17 Track17.flac | 1411 | — | 92.83 dB | — | 92.83 dB |
| sundaytimes-wind-in-the-willows/18 Track18.flac | 1411 | — | 92.18 dB | — | 92.18 dB |
| sundaytimes-wind-in-the-willows/19 Track19.flac | 1411 | — | 92.23 dB | — | 92.23 dB |
| sundaytimes-wind-in-the-willows/20 Track20.flac | 1411 | — | 92.36 dB | — | 92.36 dB |
| sundaytimes-wind-in-the-willows/21 Track21.flac | 1411 | — | 91.99 dB | — | 91.99 dB |
| sundaytimes-wind-in-the-willows/22 Track22.flac | 1411 | — | 92.09 dB | — | 92.09 dB |
| sundaytimes-wind-in-the-willows/23 Track23.flac | 1411 | — | 92.00 dB | — | 92.00 dB |
| sundaytimes-wind-in-the-willows/24 Track24.flac | 1411 | — | 91.90 dB | — | 91.90 dB |
| sundaytimes-wind-in-the-willows/25 Track25.flac | 1411 | — | 91.81 dB | — | 91.81 dB |
| sundaytimes-wind-in-the-willows/26 Track26.flac | 1411 | — | 91.79 dB | — | 91.79 dB |
| va-female-vocal-trance-2019-opus-128/Adam Ellis - Broken (& Jo Cartwright).opus | 1536 | 46.77 dB | 46.77 dB | 0.00 dB | 106.34 dB |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Hurt of intention 2019 (& Neev Kennedy).opus | 1536 | 36.64 dB | 36.64 dB | 0.00 dB | 108.24 dB |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Right back.opus | 1536 | 45.62 dB | 45.62 dB | 0.00 dB | 105.75 dB |
| va-female-vocal-trance-2019-opus-128/Alex Leavon - When I'm with you (& Julia Ross).opus | 1536 | 35.91 dB | 35.91 dB | 0.00 dB | 106.64 dB |
| va-female-vocal-trance-2019-opus-128/Ana Criado - In a thousand skies.opus | 1536 | 44.31 dB | 44.31 dB | 0.00 dB | 106.53 dB |
| va-female-vocal-trance-2019-opus-128/Blue5even - Through the barricades (& Jo Cartwright).opus | 1536 | 58.52 dB | 58.52 dB | 0.00 dB | 103.66 dB |
| va-female-vocal-trance-2019-opus-128/Braulio Stefield - See ghosts (& Victoriya).opus | 1536 | 41.19 dB | 41.19 dB | 0.00 dB | 106.67 dB |
| va-female-vocal-trance-2019-opus-128/Costa - Always (& Cathy Burton).opus | 1536 | 41.33 dB | 41.33 dB | 0.00 dB | 107.10 dB |
| va-female-vocal-trance-2019-opus-128/Delta-S - Letting go (Ikerya Project remix) (& Kate Louise Smith).opus | 1536 | 43.97 dB | 43.97 dB | 0.00 dB | 107.59 dB |
| va-female-vocal-trance-2019-opus-128/Derek Ryan - After dark (ft Melissa R. Kaplan).opus | 1536 | 68.66 dB | 68.66 dB | 0.00 dB | 103.51 dB |
| va-female-vocal-trance-2019-opus-128/Drival - Saviour (& Michele C).opus | 1536 | 52.48 dB | 52.48 dB | 0.00 dB | 105.70 dB |
| va-female-vocal-trance-2019-opus-128/F.G. Noise - Waiting for the thunder (Killing time) (& Lauren Ní Chasaide).opus | 1536 | 43.07 dB | 43.07 dB | 0.00 dB | 106.41 dB |
| va-female-vocal-trance-2019-opus-128/Kaimo K - Hold of you (Denis Kenzo remix).opus | 1536 | 46.30 dB | 46.30 dB | 0.00 dB | 107.29 dB |
| va-female-vocal-trance-2019-opus-128/Kaimo K - When you come home (Myde rework).opus | 1536 | 38.86 dB | 38.86 dB | 0.00 dB | 108.05 dB |
| va-female-vocal-trance-2019-opus-128/Karanda - Still got time (& Sarah Russell).opus | 1536 | 54.46 dB | 54.46 dB | 0.00 dB | 106.30 dB |
| va-female-vocal-trance-2019-opus-128/Limelght - Run & hide (ft Alina Renae).opus | 1536 | 47.32 dB | 47.32 dB | 0.00 dB | 105.47 dB |
| va-female-vocal-trance-2019-opus-128/Lost Witness - Sewn.opus | 1536 | 43.17 dB | 43.17 dB | 0.00 dB | 105.59 dB |
| va-female-vocal-trance-2019-opus-128/Mhammed El Alami - Warriors (& Emma Horan).opus | 1536 | 45.05 dB | 45.05 dB | 0.00 dB | 107.27 dB |
| va-female-vocal-trance-2019-opus-128/Nicholas Gunn - Older (Costa remix) (ft Alina Renae).opus | 1536 | 49.41 dB | 49.41 dB | 0.00 dB | 104.53 dB |
| va-female-vocal-trance-2019-opus-128/Nitrous Oxide - Lower than the ground (& Sarah Russell).opus | 1536 | 40.06 dB | 40.06 dB | 0.00 dB | 107.01 dB |
| va-female-vocal-trance-2019-opus-128/Passenger 75 - Heartless (& Score).opus | 1536 | 44.09 dB | 44.09 dB | 0.00 dB | 106.44 dB |
| va-female-vocal-trance-2019-opus-128/Perpetual - Innocent (ft Fisher).opus | 1536 | 39.54 dB | 39.54 dB | 0.00 dB | 107.58 dB |
| va-female-vocal-trance-2019-opus-128/Raz Nitzan - Beyond time (Aurosonic remix) (& Ellie Lawson).opus | 1536 | 48.44 dB | 48.44 dB | 0.00 dB | 106.97 dB |
| va-female-vocal-trance-2019-opus-128/Ronski Speed - Beat alive (Denis Airwave remix) (& Sarah Lynn).opus | 1536 | 40.84 dB | 40.84 dB | 0.00 dB | 107.89 dB |
| va-female-vocal-trance-2019-opus-128/Rub!k - Everglow (& Sue McLaren).opus | 1536 | 45.88 dB | 45.88 dB | 0.00 dB | 106.44 dB |
| va-female-vocal-trance-2019-opus-128/Stargazers - Be here with me (ft Katty Heath).opus | 1536 | 46.94 dB | 46.94 dB | 0.00 dB | 106.56 dB |
| va-female-vocal-trance-2019-opus-128/Stargazers - Crystalize (& Fenna Day).opus | 1536 | 48.46 dB | 48.46 dB | 0.00 dB | 106.14 dB |
| va-female-vocal-trance-2019-opus-128/The Blizzard - Always a stranger (Nitrous Oxide remix) (& Carol Lee).opus | 1536 | 43.09 dB | 43.09 dB | 0.00 dB | 105.45 dB |
| va-female-vocal-trance-2019-opus-128/The City Never Sleeps - Addicted (NyTiGen remix) (& Summer Haze).opus | 1536 | 59.75 dB | 59.75 dB | 0.00 dB | 103.75 dB |
| va-female-vocal-trance-2019-opus-128/Whiteout - The part in-between (Wilderness & A-Line remix) (& One Half Bear).opus | 1536 | 41.01 dB | 41.01 dB | 0.00 dB | 107.20 dB |

## ogg/vorbis

Measured on 84 of 84 carriers.

### Does it look like a plain ffmpeg encode?

The ffmpeg column is ffmpeg at its own defaults. Ogg Vorbis keeps the source's quality, so it differs from ffmpeg's default q3 whenever the carrier maps to another level.

| Carrier | Source samples | Samples (ffmpeg / Mist) | Zero tail (ffmpeg / Mist) | Sample format (ffmpeg / Mist) | Nominal kbps (ffmpeg / Mist) | Differs in |
|---|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 13230720 | 13230208 / 13230720 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| 03 - Disposable Heroes.mp3 | 13230720 | 13230208 / 13230720 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| 04 - No Remorse.mp3 | 13230720 | 13230208 / 13230720 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| 06 - For Whom the Bell Tolls.mp3 | 11669760 | 11670080 / 11670592 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| 07 - The Four Horsemen.mp3 | 13230720 | 13230208 / 13230720 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| 08 - Fade to Black.mp3 | 13230720 | 13230208 / 13230720 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| 09 - Seek & Destroy.mp3 | 13230720 | 13230208 / 13230720 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| 10 - Whiplash.mp3 | 13230720 | 13230208 / 13230720 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| 11 - Fight Fire with Fire.mp3 | 11631744 | 11631936 / 11632448 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| 14 - Motorbreath.mp3 | 12556800 | 12556480 / 12557248 | 30720 / 30720 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| ATB.06.20.2025/01 ATB 06-20-2025.mp3 | 13231872 | 13230848 / 13231872 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| ATB.06.20.2025/02 ATB 06-20-2025.mp3 | 13231872 | 13230848 / 13231872 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| ATB.06.20.2025/03 ATB 06-20-2025.mp3 | 13231872 | 13230848 / 13231872 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| ATB.06.20.2025/04 ATB 06-20-2025.mp3 | 13231872 | 13230848 / 13231872 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| Pillars_of_the_Sky.wav | 2880000 | 2880192 / 2880192 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| Starlight_Ascent.wav | 5760000 | 5760064 / 5760320 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| Stellar_Ascent.wav | 5760000 | 5760960 / 5760192 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side1.flac | 28803072 | 28803264 / 28803264 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side2.flac | 28803072 | 28803520 / 28803776 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | length |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t01.flac | 27650560 | 27650752 / 27651264 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | length |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t02.flac | 27998720 | 27999552 / 27998912 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | length |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t03.flac | 24209920 | 24210368 / 24210368 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t04.flac | 28803072 | 28804032 / 28803072 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | length |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t05.flac | 24574720 | 24574784 / 24574912 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | length |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t06.flac | 26830080 | 26830656 / 26830656 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t07.flac | 24695040 | 24695616 / 24695872 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | length |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t08.flac | 18909440 | 18909504 / 18909504 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t09.flac | 28803072 | 28803072 / 28803072 | 0 / 0 | fltp / fltp | 4294967 / 4294967 | — |
| sundaytimes-wind-in-the-willows/01 Track01.flac | 7939176 | 7940160 / 7940160 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/02 Track02.flac | 8334312 | 8334400 / 8335040 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/03 Track03.flac | 8187312 | 8187968 / 8187840 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/04 Track04.flac | 7616952 | 7617728 / 7617728 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/05 Track05.flac | 7703976 | 7704384 / 7704896 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/06 Track06.flac | 7795116 | 7796032 / 7795392 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/07 Track07.flac | 8786484 | 8787136 / 8787136 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/08 Track08.flac | 8306676 | 8307392 / 8307136 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/09 Track09.flac | 7848624 | 7849280 / 7849280 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/10 Track10.flac | 7849212 | 7849792 / 7849920 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/11 Track11.flac | 8371356 | 8371392 / 8371776 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/12 Track12.flac | 7808052 | 7808064 / 7808320 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/13 Track13.flac | 7792764 | 7793600 / 7793088 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/14 Track14.flac | 7705152 | 7705152 / 7705152 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/15 Track15.flac | 7733376 | 7733568 / 7733568 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/16 Track16.flac | 8332548 | 8332736 / 8332608 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/17 Track17.flac | 7836864 | 7837632 / 7837120 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/18 Track18.flac | 7783356 | 7783872 / 7783360 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/19 Track19.flac | 8134980 | 8136000 / 8135360 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/20 Track20.flac | 7904484 | 7904960 / 7905216 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/21 Track21.flac | 7577556 | 7578176 / 7578176 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/22 Track22.flac | 8433684 | 8434368 / 8433728 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/23 Track23.flac | 8167320 | 8167488 / 8168128 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/24 Track24.flac | 7443492 | 7444160 / 7444160 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| sundaytimes-wind-in-the-willows/25 Track25.flac | 7847448 | 7847872 / 7848000 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| sundaytimes-wind-in-the-willows/26 Track26.flac | 6560316 | 6560448 / 6561088 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Adam Ellis - Broken (& Jo Cartwright).opus | 7568328 | 7568448 / 7569216 | 3072 / 2048 | fltp / fltp | 112 / 256 | length, zero tail, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Hurt of intention 2019 (& Neev Kennedy).opus | 10717128 | 10717504 / 10717760 | 2048 / 2048 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Right back.opus | 10753608 | 10753472 / 10754368 | 0 / 1024 | fltp / fltp | 112 / 256 | length, zero tail, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Alex Leavon - When I'm with you (& Julia Ross).opus | 8507208 | 8507200 / 8508224 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Ana Criado - In a thousand skies.opus | 9099528 | 9099840 / 9100224 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Blue5even - Through the barricades (& Jo Cartwright).opus | 10184328 | 10185152 / 10185024 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Braulio Stefield - See ghosts (& Victoriya).opus | 9849288 | 9849152 / 9849920 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Costa - Always (& Cathy Burton).opus | 10444488 | 10444736 / 10444608 | 1024 / 1024 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Delta-S - Letting go (Ikerya Project remix) (& Kate Louise Smith).opus | 10854408 | 10854592 / 10854592 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Derek Ryan - After dark (ft Melissa R. Kaplan).opus | 9693768 | 9693632 / 9694144 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Drival - Saviour (& Michele C).opus | 9767688 | 9767872 / 9768000 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/F.G. Noise - Waiting for the thunder (Killing time) (& Lauren Ní Chasaide).opus | 9360648 | 9360320 / 9361600 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Kaimo K - Hold of you (Denis Kenzo remix).opus | 9855048 | 9855296 / 9855808 | 8192 / 8192 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Kaimo K - When you come home (Myde rework).opus | 9146568 | 9146304 / 9146816 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Karanda - Still got time (& Sarah Russell).opus | 10718088 | 10718272 / 10718400 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Limelght - Run & hide (ft Alina Renae).opus | 8370888 | 8370496 / 8371520 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Lost Witness - Sewn.opus | 10456968 | 10456256 / 10457792 | 23552 / 24576 | fltp / fltp | 112 / 256 | length, zero tail, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Mhammed El Alami - Warriors (& Emma Horan).opus | 9574728 | 9573952 / 9575232 | 7168 / 8192 | fltp / fltp | 112 / 256 | length, zero tail, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Nicholas Gunn - Older (Costa remix) (ft Alina Renae).opus | 10171848 | 10172864 / 10171968 | 1024 / 0 | fltp / fltp | 112 / 256 | length, zero tail, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Nitrous Oxide - Lower than the ground (& Sarah Russell).opus | 9736008 | 9736128 / 9736768 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Passenger 75 - Heartless (& Score).opus | 9412488 | 9412544 / 9412544 | 6144 / 6144 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Perpetual - Innocent (ft Fisher).opus | 9888648 | 9888960 / 9888960 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Raz Nitzan - Beyond time (Aurosonic remix) (& Ellie Lawson).opus | 10440648 | 10440000 / 10441152 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Ronski Speed - Beat alive (Denis Airwave remix) (& Sarah Lynn).opus | 11520648 | 11520320 / 11521344 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Rub!k - Everglow (& Sue McLaren).opus | 10451208 | 10451520 / 10451520 | 21504 / 21504 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Stargazers - Be here with me (ft Katty Heath).opus | 8598408 | 8598848 / 8598848 | 0 / 0 | fltp / fltp | 112 / 256 | nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Stargazers - Crystalize (& Fenna Day).opus | 11520648 | 11520960 / 11521088 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/The Blizzard - Always a stranger (Nitrous Oxide remix) (& Carol Lee).opus | 11103048 | 11103040 / 11103168 | 0 / 0 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/The City Never Sleeps - Addicted (NyTiGen remix) (& Summer Haze).opus | 8540808 | 8540480 / 8540864 | 6144 / 6144 | fltp / fltp | 112 / 256 | length, nominal bitrate |
| va-female-vocal-trance-2019-opus-128/Whiteout - The part in-between (Wilderness & A-Line remix) (& One Half Bear).opus | 10280328 | 10280256 / 10280512 | 6144 / 7168 | fltp / fltp | 112 / 256 | length, zero tail, nominal bitrate |

**Verdict:** ❌ differs on 79 of 84 carriers: length, nominal bitrate, zero tail.

### Can a detector tell?

Stego copy against the clean ffmpeg copy (for Ogg Vorbis, ffmpeg at the quality level Mist chose, so only the embedding differs).

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.499 (0.497–0.500) | 0.506 (0.500–0.518) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.499–0.500) | 0.512 (0.491–0.536) | 0.000 (0.000–0.003) | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.533 (0.511–0.562) | 0.584 (0.543–0.631) | 0.014 (0.004–0.034) | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.526 (0.503–0.556) | 0.532 (0.522–0.544) | 0.002 (0.001–0.004) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.524 (0.500–0.555) | 0.500 (0.464–0.530) | 0.000 (0.000–0.003) | ✅ chance | ✅ hidden (0.500) |
| key-aware | the ephemeral key in the first frame's envelope, read with the public key alone | 0.500 (0.470–0.530) | 0.500 (0.470–0.530) | 0.000 (0.000–0.002) | ✅ chance | ✅ hidden (0.500) |

**Verdict:** ⚠️ faint signal from spa (AUC 0.499).

### The embedding alone

Stego copy against Mist's own re-encode, so only the embedded changes differ.

| Detector | Looks for | Chunk AUC (95%) | File AUC (95%) | ε ≥ (nats, 95%) | Verdict | Message size |
|---|---|---|---|---|---|---|
| chi-square | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| spa | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.500–0.501) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| rs | bit overwriting (not what Mist does) | 0.500 (0.500–0.500) | 0.500 (0.499–0.502) | 0.000 (0.000–0.000) | ✅ chance | ✅ hidden (0.500) |
| hcf-com | ±1 changes, which is what Mist does | 0.502 (0.502–0.503) | 0.510 (0.508–0.513) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| classifier | anything it can learn from Mist's own output | 0.498 (0.498–0.498) | 0.491 (0.488–0.494) | 0.000 (0.000–0.000) | ⚠️ faint signal | ✅ hidden (0.500) |
| markov | how the waveform's curvature changes from sample to sample | 0.495 (0.494–0.496) | 0.487 (0.483–0.490) | 0.000 (0.000–0.001) | ⚠️ faint signal | ✅ hidden (0.500) |

### Does detection grow with audio?

File AUC (95%) against the clean ffmpeg copy on the first chunks of each carrier. A chunk is 65,536 values, about 0.74 s of 44.1 kHz stereo.

| Detector | First 40 chunks | Whole file |
|---|---|---|
| chi-square | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) |
| spa | 0.500 (0.499–0.500) | 0.506 (0.500–0.518) |
| rs | 0.511 (0.490–0.533) | 0.512 (0.491–0.536) |
| hcf-com | 0.583 (0.541–0.629) | 0.584 (0.543–0.631) |
| classifier | 0.533 (0.523–0.548) | 0.532 (0.522–0.544) |
| markov | 0.532 (0.512–0.557) | 0.500 (0.464–0.530) |

### By kind of audio

File AUC (95%) against the clean ffmpeg copy, per corpus folder. The count is carriers; few carriers means a wide interval.

| Detector | (top level) (13) | ATB.06.20.2025 (4) | disc1 (2) | hmbb2015-03-18.sbd.flac24 (9) | sundaytimes-wind-in-the-willows (26) | va-female-vocal-trance-2019-opus-128 (30) |
|---|---|---|---|---|---|---|
| chi-square | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) |
| spa | 0.538 (0.500–0.611) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.501 (0.497–0.507) |
| rs | 0.544 (0.500–0.634) | 0.375 (0.208–0.500) | 1.000 (1.000–1.000) | 0.500 (0.500–0.500) | 0.500 (0.500–0.500) | 0.499 (0.496–0.503) |
| hcf-com | 0.533 (0.472–0.580) | 0.500 (0.333–0.667) | 1.000 (1.000–1.000) | 1.000 (1.000–1.000) | 0.530 (0.521–0.546) | 0.526 (0.518–0.537) |
| classifier | 0.538 (0.514–0.579) | 0.500 (0.333–0.667) | 1.000 (1.000–1.000) | 1.000 (1.000–1.000) | 0.609 (0.576–0.658) | 0.476 (0.462–0.483) |
| markov | 0.473 (0.440–0.497) | 0.500 (0.333–0.667) | 1.000 (1.000–1.000) | 1.000 (1.000–1.000) | 0.740 (0.667–0.823) | 0.466 (0.452–0.476) |

### Does detection grow with the number of files?

AUC when the warden averages a detector's score over k files drawn at random from the corpus, stego against clean. Draws overlap, so there is no interval.

| Detector | 1 file(s) | 3 file(s) | 6 file(s) |
|---|---|---|---|
| chi-square | 0.500 | 0.500 | 0.500 |
| spa | 0.488 | 0.522 | 0.535 |
| rs | 0.482 | 0.535 | 0.550 |
| hcf-com | 0.603 | 0.691 | 0.781 |
| classifier | 0.501 | 0.552 | 0.576 |
| markov | 0.466 | 0.499 | 0.506 |

### A learned warden

A small CNN trained on the first 16 chunks of each carrier, five folds split by carrier, one score per file (`tools/cnn_warden`).

| Detector | Carriers | File AUC (95%) | ε ≥ (nats, 95%) |
|---|---|---|---|
| cnn | 84 | 0.612 (0.559–0.677) | 0.025 (0.007–0.063) |

### How much does it cost in sound?

| Measure | Mean | Worst | What it means |
|---|---|---|---|
| Plain re-encode SDR | 27.65 dB | 23.99 dB | Loss of Mist's own re-encode, with nothing embedded |
| Mist output SDR | 27.63 dB | 23.96 dB | The same, with the message embedded |
| Embedding cost | 0.02 dB | 0.04 dB | Mist's own share; target ≤ 0.3 dB, design-doc reference 0.18 dB |
| Extra error energy | +0% | +1% | Embedding cost as error added on top of a plain re-encode |
| Mist's error below the music | 51.63 dB | 44.89 dB | Stego copy against Mist's own re-encode: what embedding alone adds |

**Verdict:** ✅ Mist's own error sits 51.63 dB below the music; embedding costs 0.02 dB, within the 0.3 dB target.

### Per carrier

| Carrier | Output kbps | Plain re-encode SDR | Mist output SDR | Embedding cost | Mist's error below the music |
|---|---|---|---|---|---|
| 02 - Ride the Lightning.mp3 | 229 | 28.20 dB | 28.19 dB | 0.01 dB | 53.73 dB |
| 03 - Disposable Heroes.mp3 | 228 | 28.57 dB | 28.55 dB | 0.01 dB | 54.30 dB |
| 04 - No Remorse.mp3 | 221 | 29.57 dB | 29.56 dB | 0.01 dB | 56.89 dB |
| 06 - For Whom the Bell Tolls.mp3 | 226 | 28.37 dB | 28.37 dB | 0.01 dB | 57.20 dB |
| 07 - The Four Horsemen.mp3 | 222 | 28.79 dB | 28.78 dB | 0.01 dB | 56.74 dB |
| 08 - Fade to Black.mp3 | 226 | 29.54 dB | 29.53 dB | 0.01 dB | 55.52 dB |
| 09 - Seek & Destroy.mp3 | 227 | 29.01 dB | 29.00 dB | 0.01 dB | 55.15 dB |
| 10 - Whiplash.mp3 | 221 | 29.32 dB | 29.32 dB | 0.01 dB | 56.91 dB |
| 11 - Fight Fire with Fire.mp3 | 189 | 27.91 dB | 27.89 dB | 0.01 dB | 52.20 dB |
| 14 - Motorbreath.mp3 | 192 | 29.20 dB | 29.18 dB | 0.01 dB | 53.60 dB |
| ATB.06.20.2025/01 ATB 06-20-2025.mp3 | 268 | 32.69 dB | 32.67 dB | 0.02 dB | 55.72 dB |
| ATB.06.20.2025/02 ATB 06-20-2025.mp3 | 269 | 25.96 dB | 25.94 dB | 0.02 dB | 50.15 dB |
| ATB.06.20.2025/03 ATB 06-20-2025.mp3 | 258 | 32.90 dB | 32.88 dB | 0.01 dB | 58.16 dB |
| ATB.06.20.2025/04 ATB 06-20-2025.mp3 | 243 | 30.99 dB | 30.98 dB | 0.01 dB | 60.23 dB |
| Pillars_of_the_Sky.wav | 262 | 26.02 dB | 26.00 dB | 0.02 dB | 48.75 dB |
| Starlight_Ascent.wav | 282 | 26.09 dB | 26.07 dB | 0.02 dB | 49.31 dB |
| Stellar_Ascent.wav | 261 | 27.80 dB | 27.78 dB | 0.02 dB | 50.72 dB |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side1.flac | 332 | 30.98 dB | 30.98 dB | 0.00 dB | 68.68 dB |
| disc1/lp_music-of-glinka-and-tchaikovsky_glinka-tchaikovsky_disc1side2.flac | 295 | 30.48 dB | 30.48 dB | 0.00 dB | 71.05 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t01.flac | 244 | 29.12 dB | 29.11 dB | 0.01 dB | 56.66 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t02.flac | 218 | 28.35 dB | 28.34 dB | 0.01 dB | 54.47 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t03.flac | 257 | 29.87 dB | 29.85 dB | 0.01 dB | 55.10 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t04.flac | 220 | 29.76 dB | 29.75 dB | 0.01 dB | 57.43 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t05.flac | 271 | 28.28 dB | 28.27 dB | 0.01 dB | 54.52 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t06.flac | 226 | 27.33 dB | 27.32 dB | 0.01 dB | 52.28 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t07.flac | 273 | 27.07 dB | 27.06 dB | 0.01 dB | 52.24 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t08.flac | 224 | 26.76 dB | 26.75 dB | 0.01 dB | 52.73 dB |
| hmbb2015-03-18.sbd.flac24/hmbb2015-03-18.sbd-t09.flac | 237 | 28.52 dB | 28.51 dB | 0.01 dB | 55.01 dB |
| sundaytimes-wind-in-the-willows/01 Track01.flac | 233 | 28.56 dB | 28.53 dB | 0.02 dB | 51.74 dB |
| sundaytimes-wind-in-the-willows/02 Track02.flac | 237 | 29.33 dB | 29.30 dB | 0.02 dB | 52.42 dB |
| sundaytimes-wind-in-the-willows/03 Track03.flac | 239 | 29.55 dB | 29.53 dB | 0.02 dB | 52.96 dB |
| sundaytimes-wind-in-the-willows/04 Track04.flac | 244 | 29.52 dB | 29.50 dB | 0.01 dB | 54.50 dB |
| sundaytimes-wind-in-the-willows/05 Track05.flac | 230 | 29.20 dB | 29.18 dB | 0.02 dB | 52.78 dB |
| sundaytimes-wind-in-the-willows/06 Track06.flac | 245 | 29.59 dB | 29.57 dB | 0.02 dB | 53.20 dB |
| sundaytimes-wind-in-the-willows/07 Track07.flac | 239 | 29.11 dB | 29.09 dB | 0.02 dB | 52.72 dB |
| sundaytimes-wind-in-the-willows/08 Track08.flac | 231 | 28.73 dB | 28.71 dB | 0.02 dB | 51.85 dB |
| sundaytimes-wind-in-the-willows/09 Track09.flac | 245 | 28.61 dB | 28.59 dB | 0.02 dB | 52.00 dB |
| sundaytimes-wind-in-the-willows/10 Track10.flac | 249 | 29.27 dB | 29.25 dB | 0.02 dB | 52.87 dB |
| sundaytimes-wind-in-the-willows/11 Track11.flac | 248 | 29.16 dB | 29.13 dB | 0.02 dB | 52.12 dB |
| sundaytimes-wind-in-the-willows/12 Track12.flac | 240 | 28.96 dB | 28.94 dB | 0.02 dB | 53.01 dB |
| sundaytimes-wind-in-the-willows/13 Track13.flac | 232 | 28.92 dB | 28.90 dB | 0.02 dB | 52.43 dB |
| sundaytimes-wind-in-the-willows/14 Track14.flac | 242 | 28.21 dB | 28.19 dB | 0.02 dB | 52.04 dB |
| sundaytimes-wind-in-the-willows/15 Track15.flac | 236 | 29.10 dB | 29.07 dB | 0.03 dB | 51.78 dB |
| sundaytimes-wind-in-the-willows/16 Track16.flac | 242 | 29.12 dB | 29.10 dB | 0.02 dB | 52.36 dB |
| sundaytimes-wind-in-the-willows/17 Track17.flac | 244 | 28.05 dB | 28.03 dB | 0.02 dB | 51.85 dB |
| sundaytimes-wind-in-the-willows/18 Track18.flac | 242 | 29.18 dB | 29.16 dB | 0.02 dB | 52.75 dB |
| sundaytimes-wind-in-the-willows/19 Track19.flac | 248 | 29.15 dB | 29.13 dB | 0.02 dB | 52.82 dB |
| sundaytimes-wind-in-the-willows/20 Track20.flac | 233 | 29.35 dB | 29.33 dB | 0.02 dB | 53.59 dB |
| sundaytimes-wind-in-the-willows/21 Track21.flac | 234 | 29.24 dB | 29.21 dB | 0.02 dB | 52.48 dB |
| sundaytimes-wind-in-the-willows/22 Track22.flac | 240 | 27.36 dB | 27.33 dB | 0.03 dB | 49.92 dB |
| sundaytimes-wind-in-the-willows/23 Track23.flac | 242 | 28.76 dB | 28.74 dB | 0.02 dB | 52.11 dB |
| sundaytimes-wind-in-the-willows/24 Track24.flac | 231 | 28.44 dB | 28.42 dB | 0.01 dB | 53.68 dB |
| sundaytimes-wind-in-the-willows/25 Track25.flac | 231 | 28.97 dB | 28.96 dB | 0.02 dB | 52.70 dB |
| sundaytimes-wind-in-the-willows/26 Track26.flac | 233 | 27.31 dB | 27.29 dB | 0.02 dB | 50.05 dB |
| va-female-vocal-trance-2019-opus-128/Adam Ellis - Broken (& Jo Cartwright).opus | 263 | 26.64 dB | 26.61 dB | 0.03 dB | 49.01 dB |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Hurt of intention 2019 (& Neev Kennedy).opus | 249 | 25.93 dB | 25.91 dB | 0.03 dB | 48.02 dB |
| va-female-vocal-trance-2019-opus-128/Alex Ender - Right back.opus | 264 | 25.30 dB | 25.27 dB | 0.04 dB | 46.04 dB |
| va-female-vocal-trance-2019-opus-128/Alex Leavon - When I'm with you (& Julia Ross).opus | 255 | 25.05 dB | 25.02 dB | 0.03 dB | 46.28 dB |
| va-female-vocal-trance-2019-opus-128/Ana Criado - In a thousand skies.opus | 250 | 24.35 dB | 24.32 dB | 0.03 dB | 46.13 dB |
| va-female-vocal-trance-2019-opus-128/Blue5even - Through the barricades (& Jo Cartwright).opus | 243 | 23.99 dB | 23.96 dB | 0.03 dB | 44.89 dB |
| va-female-vocal-trance-2019-opus-128/Braulio Stefield - See ghosts (& Victoriya).opus | 248 | 25.28 dB | 25.25 dB | 0.03 dB | 46.48 dB |
| va-female-vocal-trance-2019-opus-128/Costa - Always (& Cathy Burton).opus | 245 | 26.22 dB | 26.19 dB | 0.02 dB | 48.93 dB |
| va-female-vocal-trance-2019-opus-128/Delta-S - Letting go (Ikerya Project remix) (& Kate Louise Smith).opus | 259 | 26.16 dB | 26.14 dB | 0.02 dB | 48.95 dB |
| va-female-vocal-trance-2019-opus-128/Derek Ryan - After dark (ft Melissa R. Kaplan).opus | 232 | 26.07 dB | 26.05 dB | 0.02 dB | 49.04 dB |
| va-female-vocal-trance-2019-opus-128/Drival - Saviour (& Michele C).opus | 244 | 25.82 dB | 25.79 dB | 0.03 dB | 47.45 dB |
| va-female-vocal-trance-2019-opus-128/F.G. Noise - Waiting for the thunder (Killing time) (& Lauren Ní Chasaide).opus | 251 | 25.61 dB | 25.58 dB | 0.03 dB | 47.57 dB |
| va-female-vocal-trance-2019-opus-128/Kaimo K - Hold of you (Denis Kenzo remix).opus | 256 | 24.98 dB | 24.96 dB | 0.03 dB | 46.93 dB |
| va-female-vocal-trance-2019-opus-128/Kaimo K - When you come home (Myde rework).opus | 266 | 25.83 dB | 25.79 dB | 0.03 dB | 47.07 dB |
| va-female-vocal-trance-2019-opus-128/Karanda - Still got time (& Sarah Russell).opus | 233 | 25.43 dB | 25.41 dB | 0.03 dB | 47.60 dB |
| va-female-vocal-trance-2019-opus-128/Limelght - Run & hide (ft Alina Renae).opus | 244 | 27.25 dB | 27.22 dB | 0.02 dB | 49.92 dB |
| va-female-vocal-trance-2019-opus-128/Lost Witness - Sewn.opus | 246 | 24.64 dB | 24.61 dB | 0.03 dB | 45.81 dB |
| va-female-vocal-trance-2019-opus-128/Mhammed El Alami - Warriors (& Emma Horan).opus | 248 | 25.06 dB | 25.03 dB | 0.03 dB | 46.10 dB |
| va-female-vocal-trance-2019-opus-128/Nicholas Gunn - Older (Costa remix) (ft Alina Renae).opus | 257 | 25.68 dB | 25.65 dB | 0.03 dB | 47.18 dB |
| va-female-vocal-trance-2019-opus-128/Nitrous Oxide - Lower than the ground (& Sarah Russell).opus | 244 | 25.43 dB | 25.40 dB | 0.02 dB | 47.83 dB |
| va-female-vocal-trance-2019-opus-128/Passenger 75 - Heartless (& Score).opus | 259 | 26.01 dB | 25.98 dB | 0.03 dB | 47.52 dB |
| va-female-vocal-trance-2019-opus-128/Perpetual - Innocent (ft Fisher).opus | 271 | 24.70 dB | 24.67 dB | 0.04 dB | 45.54 dB |
| va-female-vocal-trance-2019-opus-128/Raz Nitzan - Beyond time (Aurosonic remix) (& Ellie Lawson).opus | 255 | 25.17 dB | 25.14 dB | 0.03 dB | 46.59 dB |
| va-female-vocal-trance-2019-opus-128/Ronski Speed - Beat alive (Denis Airwave remix) (& Sarah Lynn).opus | 255 | 26.06 dB | 26.04 dB | 0.03 dB | 48.38 dB |
| va-female-vocal-trance-2019-opus-128/Rub!k - Everglow (& Sue McLaren).opus | 245 | 24.20 dB | 24.17 dB | 0.03 dB | 46.16 dB |
| va-female-vocal-trance-2019-opus-128/Stargazers - Be here with me (ft Katty Heath).opus | 260 | 25.76 dB | 25.73 dB | 0.03 dB | 46.90 dB |
| va-female-vocal-trance-2019-opus-128/Stargazers - Crystalize (& Fenna Day).opus | 244 | 24.64 dB | 24.61 dB | 0.03 dB | 46.55 dB |
| va-female-vocal-trance-2019-opus-128/The Blizzard - Always a stranger (Nitrous Oxide remix) (& Carol Lee).opus | 245 | 25.19 dB | 25.17 dB | 0.03 dB | 47.45 dB |
| va-female-vocal-trance-2019-opus-128/The City Never Sleeps - Addicted (NyTiGen remix) (& Summer Haze).opus | 255 | 24.11 dB | 24.08 dB | 0.03 dB | 46.09 dB |
| va-female-vocal-trance-2019-opus-128/Whiteout - The part in-between (Wilderness & A-Line remix) (& One Half Bear).opus | 261 | 27.70 dB | 27.68 dB | 0.02 dB | 50.26 dB |

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
