#ifndef MIST_AV_H
#define MIST_AV_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/*
 * Thin C boundary for libav. Stubs in cgo.c compile without FFmpeg
 * headers. The real implementation includes libavformat/avformat.h,
 * libavcodec/avcodec.h, libavutil/avutil.h and maps these types onto
 * AVFormatContext / AVCodecContext / AVPacket / AVFrame / AVIOContext.
 */

enum mist_av_err {
	MIST_AV_OK            = 0,
	MIST_AV_ERR           = -1,
	MIST_AV_EOF           = -2,
	MIST_AV_UNIMPLEMENTED = -3
};

enum mist_av_codec_id {
	MIST_AV_CODEC_NONE   = 0,
	MIST_AV_CODEC_VORBIS = 1,
	MIST_AV_CODEC_PCM    = 2,
	MIST_AV_CODEC_MP3    = 3,
	MIST_AV_CODEC_AAC    = 4
};

/* Numeric values match AVSampleFormat. */
enum mist_av_sample_fmt {
	MIST_AV_SAMPLE_FMT_NONE = -1,
	MIST_AV_SAMPLE_FMT_U8   = 0,
	MIST_AV_SAMPLE_FMT_S16  = 1,
	MIST_AV_SAMPLE_FMT_S32  = 2,
	MIST_AV_SAMPLE_FMT_FLT  = 3,
	MIST_AV_SAMPLE_FMT_DBL  = 4,
	MIST_AV_SAMPLE_FMT_U8P  = 5,
	MIST_AV_SAMPLE_FMT_S16P = 6,
	MIST_AV_SAMPLE_FMT_S32P = 7,
	MIST_AV_SAMPLE_FMT_FLTP = 8,
	MIST_AV_SAMPLE_FMT_DBLP = 9
};

typedef struct mist_av_demuxer mist_av_demuxer;
typedef struct mist_av_muxer   mist_av_muxer;
typedef struct mist_av_decoder mist_av_decoder;
typedef struct mist_av_encoder mist_av_encoder;
typedef struct mist_av_io      mist_av_io;

typedef struct mist_av_audio_info {
	int     codec_id;
	int     sample_rate;
	int     channels;
	int     sample_fmt;
	int64_t bitrate;
	int64_t duration_us;
} mist_av_audio_info;

typedef struct mist_av_packet {
	uint8_t *data;
	int      size;
	int      stream_index;
	int      flags;
	int64_t  pts;
	int64_t  dts;
	int64_t  duration;
} mist_av_packet;

typedef struct mist_av_frame {
	uint8_t *data[8];
	int      linesize[8];
	int      nb_samples;
	int      channels;
	int      sample_rate;
	int      format;
	int64_t  pts;
} mist_av_frame;

typedef int     (*mist_av_read_fn)(void *opaque, uint8_t *buf, int buf_size);
typedef int     (*mist_av_write_fn)(void *opaque, uint8_t *buf, int buf_size);
typedef int64_t (*mist_av_seek_fn)(void *opaque, int64_t offset, int whence);

int mist_av_init(void);

mist_av_demuxer *mist_av_demuxer_open(const char *url, char *errbuf, int errlen);
mist_av_demuxer *mist_av_demuxer_open_io(mist_av_io *io, char *errbuf, int errlen);
int              mist_av_demuxer_audio_info(mist_av_demuxer *d, mist_av_audio_info *info);
int              mist_av_demuxer_read(mist_av_demuxer *d, mist_av_packet *pkt);
void             mist_av_demuxer_close(mist_av_demuxer *d);

mist_av_muxer *mist_av_muxer_open(const char *url, const mist_av_audio_info *info, char *errbuf, int errlen);
mist_av_muxer *mist_av_muxer_open_io(mist_av_io *io, const mist_av_audio_info *info, char *errbuf, int errlen);
int            mist_av_muxer_write_header(mist_av_muxer *m);
int            mist_av_muxer_write(mist_av_muxer *m, const mist_av_packet *pkt);
int            mist_av_muxer_write_trailer(mist_av_muxer *m);
void           mist_av_muxer_close(mist_av_muxer *m);

mist_av_decoder *mist_av_decoder_open(const mist_av_audio_info *info, char *errbuf, int errlen);
int              mist_av_decoder_send(mist_av_decoder *dec, const mist_av_packet *pkt);
int              mist_av_decoder_receive(mist_av_decoder *dec, mist_av_frame *frame);
void             mist_av_decoder_close(mist_av_decoder *dec);

mist_av_encoder *mist_av_encoder_open(const mist_av_audio_info *info, char *errbuf, int errlen);
int              mist_av_encoder_send(mist_av_encoder *enc, const mist_av_frame *frame);
int              mist_av_encoder_receive(mist_av_encoder *enc, mist_av_packet *pkt);
void             mist_av_encoder_close(mist_av_encoder *enc);

mist_av_io *mist_av_io_new(void *opaque, mist_av_read_fn read, mist_av_write_fn write, mist_av_seek_fn seek);
void        mist_av_io_free(mist_av_io *io);

void mist_av_packet_unref(mist_av_packet *pkt);
void mist_av_frame_unref(mist_av_frame *frame);

#ifdef __cplusplus
}
#endif

#endif /* MIST_AV_H */
