/*
 * Real libavformat / libavcodec / libavutil implementation.
 * Custom AVIO uses integer handles owned by Go (see io.go); C never stores
 * a Go pointer. Packet and frame payloads are copied with av_malloc so the
 * Go side can C.GoBytes and then mist_av_*_unref. Codec IDs stay Mist-local.
 */
#include "cgo.h"

#include <errno.h>
#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavformat/version.h>
#include <libavutil/avutil.h>
#include <libavutil/channel_layout.h>
#include <libavutil/common.h>
#include <libavutil/error.h>
#include <libavutil/frame.h>
#include <libavutil/mem.h>
#include <libavutil/samplefmt.h>

extern int     mist_av_go_read(int id, uint8_t *buf, int buf_size);
extern int     mist_av_go_write(int id, uint8_t *buf, int buf_size);
extern int64_t mist_av_go_seek(int id, int64_t offset, int whence);

#define IO_BUF 4096

struct mist_av_io {
	int          handle;
	int          writable;
	uint8_t     *buf;
	AVIOContext *avio;
};

struct mist_av_demuxer {
	AVFormatContext *fmt;
	int              audio;
	mist_av_io      *io;
};

struct mist_av_muxer {
	AVFormatContext *fmt;
	mist_av_io      *io;
	int              header;
	int              trailer;
};

struct mist_av_decoder {
	AVCodecContext *ctx;
};

struct mist_av_encoder {
	AVCodecContext *ctx;
};

static void set_err(char *errbuf, int errlen, const char *msg)
{
	if (errbuf == NULL || errlen <= 0) {
		return;
	}
	strncpy(errbuf, msg, (size_t)errlen - 1);
	errbuf[errlen - 1] = '\0';
}

static void set_averr(char *errbuf, int errlen, int err, const char *ctx)
{
	char tmp[256];
	char av[128];
	av_strerror(err, av, sizeof(av));
	snprintf(tmp, sizeof(tmp), "%s: %s", ctx, av);
	set_err(errbuf, errlen, tmp);
}

static int map_ret(int err)
{
	if (err >= 0) {
		return MIST_AV_OK;
	}
	if (err == AVERROR_EOF) {
		return MIST_AV_EOF;
	}
	if (err == AVERROR(EAGAIN)) {
		return MIST_AV_EAGAIN;
	}
	return MIST_AV_ERR;
}

static enum AVCodecID to_av_codec(int id)
{
	switch (id) {
	case MIST_AV_CODEC_VORBIS:
		return AV_CODEC_ID_VORBIS;
	case MIST_AV_CODEC_MP3:
		return AV_CODEC_ID_MP3;
	case MIST_AV_CODEC_AAC:
		return AV_CODEC_ID_AAC;
	case MIST_AV_CODEC_PCM:
		return AV_CODEC_ID_PCM_S16LE;
	default:
		return AV_CODEC_ID_NONE;
	}
}

static int from_av_codec(enum AVCodecID id)
{
	switch (id) {
	case AV_CODEC_ID_VORBIS:
		return MIST_AV_CODEC_VORBIS;
	case AV_CODEC_ID_MP3:
		return MIST_AV_CODEC_MP3;
	case AV_CODEC_ID_AAC:
		return MIST_AV_CODEC_AAC;
	case AV_CODEC_ID_PCM_S16LE:
		return MIST_AV_CODEC_PCM;
	default:
		return MIST_AV_CODEC_NONE;
	}
}

static int copy_packet(const AVPacket *src, mist_av_packet *dst)
{
	memset(dst, 0, sizeof(*dst));
	if (src->size > 0 && src->data != NULL) {
		dst->data = av_malloc((size_t)src->size);
		if (dst->data == NULL) {
			return AVERROR(ENOMEM);
		}
		memcpy(dst->data, src->data, (size_t)src->size);
	}
	dst->size = src->size;
	dst->stream_index = src->stream_index;
	dst->flags = src->flags;
	dst->pts = src->pts;
	dst->dts = src->dts;
	dst->duration = src->duration;
	return 0;
}

/*
 * resolve_codec_id prefers the id libav itself reported for the stream, so
 * any format the installed FFmpeg understands can be decoded. The
 * Mist-local id is the fallback for infos built by hand (encoder open,
 * tests), which only ever name codecs Mist knows.
 */
static enum AVCodecID resolve_codec_id(const mist_av_audio_info *info)
{
	if (info->native_codec_id != AV_CODEC_ID_NONE) {
		return (enum AVCodecID)info->native_codec_id;
	}
	return to_av_codec(info->codec_id);
}

int mist_av_can_decode(const mist_av_audio_info *info)
{
	if (info == NULL) {
		return 0;
	}
	return avcodec_find_decoder(resolve_codec_id(info)) != NULL;
}

int mist_av_is_lossless(int native_codec_id)
{
	const AVCodecDescriptor *d = avcodec_descriptor_get((enum AVCodecID)native_codec_id);
	return d != NULL && (d->props & AV_CODEC_PROP_LOSSLESS) != 0;
}

static void copy_token(char *dst, size_t dstlen, const char *src)
{
	size_t i = 0;
	if (src == NULL) {
		dst[0] = '\0';
		return;
	}
	for (; src[i] != '\0' && src[i] != ',' && i + 1 < dstlen; i++) {
		dst[i] = src[i];
	}
	dst[i] = '\0';
}

/*
 * muxer_for finds a container for id. An audio-only muxer that names id as
 * its own default is the natural home (wav for pcm_s16le, caf for alac);
 * anything else that merely accepts the codec is the fallback.
 */
static const AVOutputFormat *muxer_for(enum AVCodecID id)
{
	const AVOutputFormat *f = NULL;
	void                 *it = NULL;
	while ((f = av_muxer_iterate(&it)) != NULL) {
		if (av_guess_codec(f, NULL, NULL, NULL, AVMEDIA_TYPE_AUDIO) == id &&
		    f->video_codec == AV_CODEC_ID_NONE) {
			return f;
		}
	}
	it = NULL;
	while ((f = av_muxer_iterate(&it)) != NULL) {
		if (avformat_query_codec(f, id, FF_COMPLIANCE_NORMAL) == 1) {
			return f;
		}
	}
	return NULL;
}

/*
 * find_audio_encoder takes either name libav knows a codec by: the
 * encoder's own ("dca", "libvorbis") or the codec's ("dts", "vorbis").
 * The two differ often enough that accepting only one would make a name
 * Mist itself printed unusable as --out-codec.
 */
static const AVCodec *find_audio_encoder(const char *name)
{
	const AVCodec *enc = avcodec_find_encoder_by_name(name);
	if (enc == NULL) {
		const AVCodecDescriptor *d = avcodec_descriptor_get_by_name(name);
		if (d != NULL) {
			enc = avcodec_find_encoder(d->id);
		}
	}
	if (enc == NULL || enc->type != AVMEDIA_TYPE_AUDIO) {
		return NULL;
	}
	return enc;
}

static int fill_format(const AVOutputFormat *ofmt, enum AVCodecID id, mist_av_format *out)
{
	memset(out, 0, sizeof(*out));
	copy_token(out->container, sizeof(out->container), ofmt->name);
	snprintf(out->codec_name, sizeof(out->codec_name), "%s", avcodec_get_name(id));
	copy_token(out->ext, sizeof(out->ext), ofmt->extensions);
	if (out->ext[0] == '\0') {
		copy_token(out->ext, sizeof(out->ext), ofmt->name);
	}
	out->codec_id = (int)id;
	out->lossless = mist_av_is_lossless((int)id);
	return MIST_AV_OK;
}

int mist_av_format_find(const char *name, const char *codec, mist_av_format *out)
{
	if (name == NULL || out == NULL || name[0] == '\0') {
		return MIST_AV_ERR;
	}
	/* Passing name as both lets libav take it as -f or as an output path. */
	const AVOutputFormat *ofmt = av_guess_format(name, name, NULL);

	if (codec != NULL && codec[0] != '\0') {
		const AVCodec *enc = find_audio_encoder(codec);
		if (enc == NULL) {
			return MIST_AV_ERR;
		}
		if (ofmt == NULL || avformat_query_codec(ofmt, enc->id, FF_COMPLIANCE_NORMAL) != 1) {
			ofmt = muxer_for(enc->id);
		}
		return ofmt == NULL ? MIST_AV_ERR : fill_format(ofmt, enc->id, out);
	}
	if (ofmt != NULL) {
		enum AVCodecID id = av_guess_codec(ofmt, NULL, NULL, NULL, AVMEDIA_TYPE_AUDIO);
		if (id != AV_CODEC_ID_NONE && avcodec_find_encoder(id) != NULL) {
			return fill_format(ofmt, id, out);
		}
	}
	const AVCodec *enc = find_audio_encoder(name);
	if (enc == NULL) {
		return MIST_AV_ERR;
	}
	ofmt = muxer_for(enc->id);
	return ofmt == NULL ? MIST_AV_ERR : fill_format(ofmt, enc->id, out);
}

/*
 * mist_av_format_list writes every writable target as a newline-separated
 * list: container names first, since those are what a caller would type,
 * then the encoder names that no container is default for. Lossy codecs
 * are left to the Go layer to filter — this reports what libav can write.
 */
int mist_av_format_list(char *buf, int buflen)
{
	int                   n = 0;
	const AVOutputFormat *f = NULL;
	void                 *it = NULL;
	const AVCodec        *c = NULL;

	if (buf == NULL || buflen <= 0) {
		return MIST_AV_ERR;
	}
	buf[0] = '\0';
	while ((f = av_muxer_iterate(&it)) != NULL) {
		enum AVCodecID id = av_guess_codec(f, NULL, NULL, NULL, AVMEDIA_TYPE_AUDIO);
		char           name[32];
		/*
		 * No extension means no file: the checksum and null muxers are
		 * writable audio targets libav will happily accept, and useless
		 * as carriers.
		 */
		if (id == AV_CODEC_ID_NONE || avcodec_find_encoder(id) == NULL || f->extensions == NULL) {
			continue;
		}
		copy_token(name, sizeof(name), f->name);
		n += snprintf(buf + n, (size_t)(buflen - n), "%s\t%d\n", name, (int)id);
		if (n >= buflen) {
			return MIST_AV_OK;
		}
	}
	it = NULL;
	while ((c = av_codec_iterate(&it)) != NULL) {
		if (c->type != AVMEDIA_TYPE_AUDIO || !av_codec_is_encoder(c) ||
		    !mist_av_is_lossless((int)c->id) || muxer_for(c->id) == NULL) {
			continue;
		}
		n += snprintf(buf + n, (size_t)(buflen - n), "%s\t%d\n", c->name, (int)c->id);
		if (n >= buflen) {
			return MIST_AV_OK;
		}
	}
	return MIST_AV_OK;
}

static int fill_info_from_par(const AVCodecParameters *par, int64_t duration_us, mist_av_audio_info *info)
{
	memset(info, 0, sizeof(*info));
	info->codec_id = from_av_codec(par->codec_id);
	info->native_codec_id = (int)par->codec_id;
	snprintf(info->codec_name, sizeof(info->codec_name), "%s", avcodec_get_name(par->codec_id));
	info->sample_rate = par->sample_rate;
	info->channels = par->ch_layout.nb_channels;
	info->sample_fmt = par->format;
	info->bitrate = par->bit_rate;
	info->duration_us = duration_us;
	if (par->extradata_size > 0 && par->extradata != NULL) {
		info->extradata = av_malloc((size_t)par->extradata_size);
		if (info->extradata == NULL) {
			return AVERROR(ENOMEM);
		}
		memcpy(info->extradata, par->extradata, (size_t)par->extradata_size);
		info->extradata_size = par->extradata_size;
	}
	return 0;
}

static int apply_info_to_par(AVCodecParameters *par, const mist_av_audio_info *info)
{
	par->codec_type = AVMEDIA_TYPE_AUDIO;
	par->codec_id = resolve_codec_id(info);
	par->sample_rate = info->sample_rate;
	par->format = info->sample_fmt;
	par->bit_rate = info->bitrate;
	/*
	 * Depth, twice: raw-PCM containers size their blocks from the coded
	 * bits, and a codec that stores its own depth (TTA) writes the raw
	 * bits into its header and refuses to be read back without them.
	 */
	par->bits_per_coded_sample = av_get_bits_per_sample(par->codec_id);
	if (info->sample_fmt >= 0) {
		par->bits_per_raw_sample = av_get_bytes_per_sample(info->sample_fmt) * 8;
	}
	av_channel_layout_default(&par->ch_layout, info->channels > 0 ? info->channels : 2);
	if (info->extradata_size > 0 && info->extradata != NULL) {
		par->extradata = av_mallocz((size_t)info->extradata_size + AV_INPUT_BUFFER_PADDING_SIZE);
		if (par->extradata == NULL) {
			return AVERROR(ENOMEM);
		}
		memcpy(par->extradata, info->extradata, (size_t)info->extradata_size);
		par->extradata_size = info->extradata_size;
	}
	return 0;
}

static int read_cb(void *opaque, uint8_t *buf, int buf_size)
{
	int n = mist_av_go_read((int)(intptr_t)opaque, buf, buf_size);
	if (n == 0) {
		return AVERROR_EOF;
	}
	if (n < 0) {
		return AVERROR(EIO);
	}
	return n;
}

#if LIBAVFORMAT_VERSION_MAJOR >= 61
static int write_cb(void *opaque, const uint8_t *buf, int buf_size)
#else
static int write_cb(void *opaque, uint8_t *buf, int buf_size)
#endif
{
	int n = mist_av_go_write((int)(intptr_t)opaque, (uint8_t *)buf, buf_size);
	if (n < 0) {
		return AVERROR(EIO);
	}
	return n;
}

static int64_t seek_cb(void *opaque, int64_t offset, int whence)
{
	return mist_av_go_seek((int)(intptr_t)opaque, offset, whence);
}

/*
 * Failures reach Go through return codes and errbuf, so libav's own chatter
 * is redundant and lands in the middle of the CLI's output: cover art in a
 * normal MP3 makes it complain about the image stream Mist never looks at.
 * Quiet by default, but MIST_AV_LOG turns it back up for debugging, which is
 * the only way to see what the codec layer is doing.
 */
static int log_level_from_env(void)
{
	const char *want = getenv("MIST_AV_LOG");
	if (want == NULL || want[0] == '\0') {
		return AV_LOG_FATAL;
	}
	static const struct { const char *name; int level; } levels[] = {
		{ "quiet",   AV_LOG_QUIET   },
		{ "panic",   AV_LOG_PANIC   },
		{ "fatal",   AV_LOG_FATAL   },
		{ "error",   AV_LOG_ERROR   },
		{ "warning", AV_LOG_WARNING },
		{ "info",    AV_LOG_INFO    },
		{ "verbose", AV_LOG_VERBOSE },
		{ "debug",   AV_LOG_DEBUG   },
		{ "trace",   AV_LOG_TRACE   },
	};
	for (size_t i = 0; i < sizeof(levels) / sizeof(levels[0]); i++) {
		if (strcmp(want, levels[i].name) == 0) {
			return levels[i].level;
		}
	}
	return AV_LOG_FATAL;
}

int mist_av_init(void)
{
	avformat_network_init();
	av_log_set_level(log_level_from_env());
	return MIST_AV_OK;
}

void mist_av_free(void *p)
{
	av_free(p);
}

mist_av_io *mist_av_io_new(int handle, int writable)
{
	mist_av_io *io = av_mallocz(sizeof(*io));
	if (io == NULL) {
		return NULL;
	}
	io->handle = handle;
	io->writable = writable;
	io->buf = av_malloc(IO_BUF);
	if (io->buf == NULL) {
		av_free(io);
		return NULL;
	}
	io->avio = avio_alloc_context(
		io->buf, IO_BUF, writable, (void *)(intptr_t)handle,
		writable ? NULL : read_cb,
		writable ? write_cb : NULL,
		seek_cb);
	if (io->avio == NULL) {
		av_free(io->buf);
		av_free(io);
		return NULL;
	}
	return io;
}

void mist_av_io_free(mist_av_io *io)
{
	if (io == NULL) {
		return;
	}
	if (io->avio != NULL) {
		av_freep(&io->avio->buffer);
		avio_context_free(&io->avio);
		io->buf = NULL;
	}
	av_free(io);
}

static int find_audio(AVFormatContext *fmt)
{
	int idx = av_find_best_stream(fmt, AVMEDIA_TYPE_AUDIO, -1, -1, NULL, 0);
	return idx;
}

mist_av_demuxer *mist_av_demuxer_open(const char *url, char *errbuf, int errlen)
{
	mist_av_demuxer *d = av_mallocz(sizeof(*d));
	if (d == NULL) {
		set_err(errbuf, errlen, "oom");
		return NULL;
	}
	int err = avformat_open_input(&d->fmt, url, NULL, NULL);
	if (err < 0) {
		set_averr(errbuf, errlen, err, "open");
		av_free(d);
		return NULL;
	}
	err = avformat_find_stream_info(d->fmt, NULL);
	if (err < 0) {
		set_averr(errbuf, errlen, err, "stream info");
		avformat_close_input(&d->fmt);
		av_free(d);
		return NULL;
	}
	d->audio = find_audio(d->fmt);
	if (d->audio < 0) {
		set_err(errbuf, errlen, "no audio stream");
		avformat_close_input(&d->fmt);
		av_free(d);
		return NULL;
	}
	return d;
}

mist_av_demuxer *mist_av_demuxer_open_io(mist_av_io *io, char *errbuf, int errlen)
{
	if (io == NULL || io->avio == NULL) {
		set_err(errbuf, errlen, "nil io");
		return NULL;
	}
	mist_av_demuxer *d = av_mallocz(sizeof(*d));
	if (d == NULL) {
		set_err(errbuf, errlen, "oom");
		return NULL;
	}
	d->fmt = avformat_alloc_context();
	if (d->fmt == NULL) {
		set_err(errbuf, errlen, "oom");
		av_free(d);
		return NULL;
	}
	d->fmt->pb = io->avio;
	d->fmt->flags |= AVFMT_FLAG_CUSTOM_IO;
	d->io = io;
	int err = avformat_open_input(&d->fmt, NULL, NULL, NULL);
	if (err < 0) {
		set_averr(errbuf, errlen, err, "open io");
		avformat_free_context(d->fmt);
		av_free(d);
		return NULL;
	}
	err = avformat_find_stream_info(d->fmt, NULL);
	if (err < 0) {
		set_averr(errbuf, errlen, err, "stream info");
		avformat_close_input(&d->fmt);
		av_free(d);
		return NULL;
	}
	d->audio = find_audio(d->fmt);
	if (d->audio < 0) {
		set_err(errbuf, errlen, "no audio stream");
		avformat_close_input(&d->fmt);
		av_free(d);
		return NULL;
	}
	return d;
}

int mist_av_demuxer_audio_info(mist_av_demuxer *d, mist_av_audio_info *info)
{
	if (d == NULL || d->fmt == NULL || info == NULL || d->audio < 0) {
		return MIST_AV_ERR;
	}
	AVStream *st = d->fmt->streams[d->audio];
	int64_t dur = 0;
	if (st->duration > 0 && st->time_base.den > 0) {
		dur = av_rescale_q(st->duration, st->time_base, AV_TIME_BASE_Q);
	} else if (d->fmt->duration > 0) {
		dur = d->fmt->duration;
	}
	int err = fill_info_from_par(st->codecpar, dur, info);
	return err < 0 ? MIST_AV_ERR : MIST_AV_OK;
}

int mist_av_demuxer_read(mist_av_demuxer *d, mist_av_packet *pkt)
{
	if (d == NULL || d->fmt == NULL || pkt == NULL) {
		return MIST_AV_ERR;
	}
	AVPacket *avpkt = av_packet_alloc();
	if (avpkt == NULL) {
		return MIST_AV_ERR;
	}
	for (;;) {
		int err = av_read_frame(d->fmt, avpkt);
		if (err < 0) {
			av_packet_free(&avpkt);
			return map_ret(err);
		}
		if (avpkt->stream_index == d->audio) {
			err = copy_packet(avpkt, pkt);
			av_packet_free(&avpkt);
			return err < 0 ? MIST_AV_ERR : MIST_AV_OK;
		}
		av_packet_unref(avpkt);
	}
}

void mist_av_demuxer_close(mist_av_demuxer *d)
{
	if (d == NULL) {
		return;
	}
	if (d->fmt != NULL) {
		if (d->io != NULL) {
			d->fmt->pb = NULL;
		}
		avformat_close_input(&d->fmt);
	}
	av_free(d);
}

static mist_av_muxer *muxer_alloc(const char *url, const mist_av_audio_info *info, mist_av_io *io, char *errbuf, int errlen)
{
	mist_av_muxer *m = av_mallocz(sizeof(*m));
	if (m == NULL) {
		set_err(errbuf, errlen, "oom");
		return NULL;
	}
	const char *container = info->container[0] != '\0' ? info->container : "ogg";
	int         err = avformat_alloc_output_context2(&m->fmt, NULL, container, url);
	if (err < 0 || m->fmt == NULL) {
		set_averr(errbuf, errlen, err, "alloc container");
		av_free(m);
		return NULL;
	}
	AVStream *st = avformat_new_stream(m->fmt, NULL);
	if (st == NULL) {
		set_err(errbuf, errlen, "new stream");
		avformat_free_context(m->fmt);
		av_free(m);
		return NULL;
	}
	err = apply_info_to_par(st->codecpar, info);
	if (err < 0) {
		set_err(errbuf, errlen, "codecpar");
		avformat_free_context(m->fmt);
		av_free(m);
		return NULL;
	}
	if (io != NULL) {
		m->fmt->pb = io->avio;
		m->fmt->flags |= AVFMT_FLAG_CUSTOM_IO;
		m->io = io;
	} else if (url != NULL && !(m->fmt->oformat->flags & AVFMT_NOFILE)) {
		err = avio_open(&m->fmt->pb, url, AVIO_FLAG_WRITE);
		if (err < 0) {
			set_averr(errbuf, errlen, err, "avio_open");
			avformat_free_context(m->fmt);
			av_free(m);
			return NULL;
		}
	}
	return m;
}

mist_av_muxer *mist_av_muxer_open(const char *url, const mist_av_audio_info *info, char *errbuf, int errlen)
{
	if (info == NULL) {
		set_err(errbuf, errlen, "nil info");
		return NULL;
	}
	return muxer_alloc(url, info, NULL, errbuf, errlen);
}

mist_av_muxer *mist_av_muxer_open_io(mist_av_io *io, const mist_av_audio_info *info, char *errbuf, int errlen)
{
	if (info == NULL || io == NULL) {
		set_err(errbuf, errlen, "nil info/io");
		return NULL;
	}
	return muxer_alloc(NULL, info, io, errbuf, errlen);
}

int mist_av_muxer_write_header(mist_av_muxer *m)
{
	if (m == NULL || m->fmt == NULL) {
		return MIST_AV_ERR;
	}
	int err = avformat_write_header(m->fmt, NULL);
	if (err < 0) {
		return MIST_AV_ERR;
	}
	m->header = 1;
	return MIST_AV_OK;
}

int mist_av_muxer_write(mist_av_muxer *m, const mist_av_packet *pkt)
{
	if (m == NULL || m->fmt == NULL || pkt == NULL) {
		return MIST_AV_ERR;
	}
	AVPacket *avpkt = av_packet_alloc();
	if (avpkt == NULL) {
		return MIST_AV_ERR;
	}
	if (pkt->size > 0 && pkt->data != NULL) {
		if (av_new_packet(avpkt, pkt->size) < 0) {
			av_packet_free(&avpkt);
			return MIST_AV_ERR;
		}
		memcpy(avpkt->data, pkt->data, (size_t)pkt->size);
	}
	avpkt->stream_index = 0;
	avpkt->flags = pkt->flags;
	avpkt->pts = pkt->pts;
	avpkt->dts = pkt->dts;
	avpkt->duration = pkt->duration;
	int err = av_interleaved_write_frame(m->fmt, avpkt);
	av_packet_free(&avpkt);
	return err < 0 ? MIST_AV_ERR : MIST_AV_OK;
}

int mist_av_muxer_write_trailer(mist_av_muxer *m)
{
	if (m == NULL || m->fmt == NULL) {
		return MIST_AV_ERR;
	}
	if (!m->header || m->trailer) {
		return MIST_AV_OK;
	}
	int err = av_write_trailer(m->fmt);
	m->trailer = 1;
	return err < 0 ? MIST_AV_ERR : MIST_AV_OK;
}

void mist_av_muxer_close(mist_av_muxer *m)
{
	if (m == NULL) {
		return;
	}
	if (m->fmt != NULL) {
		if (m->header && !m->trailer) {
			av_write_trailer(m->fmt);
			m->trailer = 1;
		}
		if (m->io != NULL) {
			m->fmt->pb = NULL;
		} else if (m->fmt->pb != NULL && !(m->fmt->oformat->flags & AVFMT_NOFILE)) {
			avio_closep(&m->fmt->pb);
		}
		avformat_free_context(m->fmt);
	}
	av_free(m);
}

mist_av_decoder *mist_av_decoder_open(const mist_av_audio_info *info, char *errbuf, int errlen)
{
	if (info == NULL) {
		set_err(errbuf, errlen, "nil info");
		return NULL;
	}
	const AVCodec *codec = avcodec_find_decoder(resolve_codec_id(info));
	if (codec == NULL) {
		set_err(errbuf, errlen, "decoder not found");
		return NULL;
	}
	mist_av_decoder *d = av_mallocz(sizeof(*d));
	if (d == NULL) {
		set_err(errbuf, errlen, "oom");
		return NULL;
	}
	d->ctx = avcodec_alloc_context3(codec);
	if (d->ctx == NULL) {
		set_err(errbuf, errlen, "oom");
		av_free(d);
		return NULL;
	}
	d->ctx->sample_rate = info->sample_rate;
	d->ctx->sample_fmt = info->sample_fmt >= 0 ? info->sample_fmt : AV_SAMPLE_FMT_FLTP;
	av_channel_layout_default(&d->ctx->ch_layout, info->channels > 0 ? info->channels : 2);
	if (info->extradata_size > 0 && info->extradata != NULL) {
		d->ctx->extradata = av_mallocz((size_t)info->extradata_size + AV_INPUT_BUFFER_PADDING_SIZE);
		if (d->ctx->extradata == NULL) {
			set_err(errbuf, errlen, "oom");
			avcodec_free_context(&d->ctx);
			av_free(d);
			return NULL;
		}
		memcpy(d->ctx->extradata, info->extradata, (size_t)info->extradata_size);
		d->ctx->extradata_size = info->extradata_size;
	}
	int err = avcodec_open2(d->ctx, codec, NULL);
	if (err < 0) {
		set_averr(errbuf, errlen, err, "open decoder");
		avcodec_free_context(&d->ctx);
		av_free(d);
		return NULL;
	}
	return d;
}

int mist_av_decoder_send(mist_av_decoder *dec, const mist_av_packet *pkt)
{
	if (dec == NULL || dec->ctx == NULL) {
		return MIST_AV_ERR;
	}
	if (pkt == NULL || pkt->size == 0) {
		return map_ret(avcodec_send_packet(dec->ctx, NULL));
	}
	AVPacket *avpkt = av_packet_alloc();
	if (avpkt == NULL) {
		return MIST_AV_ERR;
	}
	if (av_new_packet(avpkt, pkt->size) < 0) {
		av_packet_free(&avpkt);
		return MIST_AV_ERR;
	}
	memcpy(avpkt->data, pkt->data, (size_t)pkt->size);
	avpkt->pts = pkt->pts;
	avpkt->dts = pkt->dts;
	avpkt->duration = pkt->duration;
	avpkt->flags = pkt->flags;
	int err = avcodec_send_packet(dec->ctx, avpkt);
	av_packet_free(&avpkt);
	return map_ret(err);
}

int mist_av_decoder_receive(mist_av_decoder *dec, mist_av_frame *frame)
{
	if (dec == NULL || dec->ctx == NULL || frame == NULL) {
		return MIST_AV_ERR;
	}
	memset(frame, 0, sizeof(*frame));
	AVFrame *fr = av_frame_alloc();
	if (fr == NULL) {
		return MIST_AV_ERR;
	}
	int err = avcodec_receive_frame(dec->ctx, fr);
	if (err < 0) {
		av_frame_free(&fr);
		return map_ret(err);
	}
	int ch = fr->ch_layout.nb_channels;
	int planar = av_sample_fmt_is_planar(fr->format);
	int planes = planar ? ch : 1;
	int bps = av_get_bytes_per_sample(fr->format);
	int plane_sz = planar ? (fr->nb_samples * bps) : (fr->nb_samples * bps * ch);
	for (int i = 0; i < planes && i < 8; i++) {
		frame->data[i] = av_malloc((size_t)plane_sz);
		if (frame->data[i] == NULL) {
			mist_av_frame_unref(frame);
			av_frame_free(&fr);
			return MIST_AV_ERR;
		}
		memcpy(frame->data[i], fr->data[i], (size_t)plane_sz);
		frame->linesize[i] = plane_sz;
	}
	frame->nb_samples = fr->nb_samples;
	frame->channels = ch;
	frame->sample_rate = fr->sample_rate;
	frame->format = fr->format;
	frame->pts = fr->pts;
	av_frame_free(&fr);
	return MIST_AV_OK;
}

void mist_av_decoder_close(mist_av_decoder *dec)
{
	if (dec == NULL) {
		return;
	}
	avcodec_free_context(&dec->ctx);
	av_free(dec);
}

/*
 * Sample formats in the order Mist wants them. Stego bits sit in the LSB
 * of the integer grid the encoder quantizes to, so a narrow grid is a
 * shallower carrier: s16 first keeps a lossless target at CD depth rather
 * than the u8 some encoders (wavpack) happen to list first. Vorbis offers
 * only fltp and lands there whatever this list says.
 */
static const enum AVSampleFormat fmt_pref[] = {
	AV_SAMPLE_FMT_S16, AV_SAMPLE_FMT_S16P,
	AV_SAMPLE_FMT_S32, AV_SAMPLE_FMT_S32P,
	AV_SAMPLE_FMT_FLT, AV_SAMPLE_FMT_FLTP,
	AV_SAMPLE_FMT_DBL, AV_SAMPLE_FMT_DBLP,
	AV_SAMPLE_FMT_U8,  AV_SAMPLE_FMT_U8P,
};

static const enum AVSampleFormat *supported_fmts(const AVCodec *codec, AVCodecContext *ctx)
{
#if LIBAVCODEC_VERSION_INT >= AV_VERSION_INT(61, 13, 100)
	const enum AVSampleFormat *fmts = NULL;
	if (avcodec_get_supported_config(ctx, codec, AV_CODEC_CONFIG_SAMPLE_FORMAT, 0,
	                                 (const void **)&fmts, NULL) < 0) {
		return NULL;
	}
	return fmts;
#else
	(void)ctx;
	return codec->sample_fmts;
#endif
}

static enum AVSampleFormat pick_sample_fmt(const AVCodec *codec, AVCodecContext *ctx)
{
	const enum AVSampleFormat *have = supported_fmts(codec, ctx);
	if (have == NULL) {
		return AV_SAMPLE_FMT_FLTP;
	}
	for (size_t i = 0; i < sizeof(fmt_pref) / sizeof(fmt_pref[0]); i++) {
		for (int j = 0; have[j] != AV_SAMPLE_FMT_NONE; j++) {
			if (have[j] == fmt_pref[i]) {
				return fmt_pref[i];
			}
		}
	}
	return have[0] != AV_SAMPLE_FMT_NONE ? have[0] : AV_SAMPLE_FMT_FLTP;
}

mist_av_encoder *mist_av_encoder_open(const mist_av_audio_info *info, char *errbuf, int errlen)
{
	if (info == NULL) {
		set_err(errbuf, errlen, "nil info");
		return NULL;
	}
	const AVCodec *codec = avcodec_find_encoder(resolve_codec_id(info));
	if (codec == NULL) {
		set_err(errbuf, errlen, "encoder not found");
		return NULL;
	}
	mist_av_encoder *e = av_mallocz(sizeof(*e));
	if (e == NULL) {
		set_err(errbuf, errlen, "oom");
		return NULL;
	}
	e->ctx = avcodec_alloc_context3(codec);
	if (e->ctx == NULL) {
		set_err(errbuf, errlen, "oom");
		av_free(e);
		return NULL;
	}
	e->ctx->sample_rate = info->sample_rate > 0 ? info->sample_rate : 44100;
	e->ctx->sample_fmt = pick_sample_fmt(codec, e->ctx);
	/* A lossless encoder ignores bit_rate; forcing one makes it complain. */
	if (info->bitrate > 0) {
		e->ctx->bit_rate = info->bitrate;
	}
	av_channel_layout_default(&e->ctx->ch_layout, info->channels > 0 ? info->channels : 2);
	e->ctx->time_base = (AVRational){1, e->ctx->sample_rate};
	int err = avcodec_open2(e->ctx, codec, NULL);
	if (err < 0) {
		set_averr(errbuf, errlen, err, "open encoder");
		avcodec_free_context(&e->ctx);
		av_free(e);
		return NULL;
	}
	return e;
}

int mist_av_encoder_info(mist_av_encoder *enc, mist_av_audio_info *info)
{
	if (enc == NULL || enc->ctx == NULL || info == NULL) {
		return MIST_AV_ERR;
	}
	memset(info, 0, sizeof(*info));
	info->codec_id = from_av_codec(enc->ctx->codec_id);
	info->native_codec_id = (int)enc->ctx->codec_id;
	snprintf(info->codec_name, sizeof(info->codec_name), "%s", avcodec_get_name(enc->ctx->codec_id));
	info->sample_rate = enc->ctx->sample_rate;
	info->channels = enc->ctx->ch_layout.nb_channels;
	info->sample_fmt = enc->ctx->sample_fmt;
	info->bitrate = enc->ctx->bit_rate;
	if (enc->ctx->extradata_size > 0 && enc->ctx->extradata != NULL) {
		info->extradata = av_malloc((size_t)enc->ctx->extradata_size);
		if (info->extradata == NULL) {
			return MIST_AV_ERR;
		}
		memcpy(info->extradata, enc->ctx->extradata, (size_t)enc->ctx->extradata_size);
		info->extradata_size = enc->ctx->extradata_size;
	}
	info->frame_size = enc->ctx->frame_size;
	return MIST_AV_OK;
}

/*
 * store_sample writes one float sample in the encoder's own format. The
 * scale factors are powers of two and match the decode side in sample.go
 * exactly, which is what lets a bit placed in a sample's LSB survive the
 * encode/decode round trip of a lossless codec.
 */
static void store_sample(uint8_t *dst, int i, int fmt, float v)
{
	switch (fmt) {
	case AV_SAMPLE_FMT_U8:
	case AV_SAMPLE_FMT_U8P:
		dst[i] = (uint8_t)(av_clip(lrintf(v * 128.0f), -128, 127) + 128);
		break;
	case AV_SAMPLE_FMT_S16:
	case AV_SAMPLE_FMT_S16P:
		((int16_t *)dst)[i] = (int16_t)av_clip(lrintf(v * 32768.0f), -32768, 32767);
		break;
	case AV_SAMPLE_FMT_S32:
	case AV_SAMPLE_FMT_S32P:
		((int32_t *)dst)[i] = (int32_t)av_clipl_int32(llrint((double)v * 2147483648.0));
		break;
	case AV_SAMPLE_FMT_FLT:
	case AV_SAMPLE_FMT_FLTP:
		((float *)dst)[i] = v;
		break;
	case AV_SAMPLE_FMT_DBL:
	case AV_SAMPLE_FMT_DBLP:
		((double *)dst)[i] = (double)v;
		break;
	default:
		break;
	}
}

int mist_av_encoder_send_flt(mist_av_encoder *enc, float **planes, int nplanes, int nb_samples, int64_t pts)
{
	if (enc == NULL || enc->ctx == NULL || planes == NULL || nplanes <= 0) {
		return MIST_AV_ERR;
	}
	AVFrame *fr = av_frame_alloc();
	if (fr == NULL) {
		return MIST_AV_ERR;
	}
	fr->nb_samples = nb_samples;
	fr->format = enc->ctx->sample_fmt;
	fr->sample_rate = enc->ctx->sample_rate;
	fr->pts = pts;
	av_channel_layout_copy(&fr->ch_layout, &enc->ctx->ch_layout);
	int err = av_frame_get_buffer(fr, 0);
	if (err < 0) {
		av_frame_free(&fr);
		return MIST_AV_ERR;
	}
	int ch = enc->ctx->ch_layout.nb_channels;
	int fmt = enc->ctx->sample_fmt;
	int planar = av_sample_fmt_is_planar(fmt);
	for (int c = 0; c < ch; c++) {
		const float *src = planes[c < nplanes ? c : nplanes - 1];
		for (int i = 0; i < nb_samples; i++) {
			if (planar) {
				store_sample(fr->data[c], i, fmt, src[i]);
			} else {
				store_sample(fr->data[0], i * ch + c, fmt, src[i]);
			}
		}
	}
	err = avcodec_send_frame(enc->ctx, fr);
	av_frame_free(&fr);
	return map_ret(err);
}

int mist_av_encoder_flush(mist_av_encoder *enc)
{
	if (enc == NULL || enc->ctx == NULL) {
		return MIST_AV_ERR;
	}
	return map_ret(avcodec_send_frame(enc->ctx, NULL));
}

int mist_av_encoder_receive(mist_av_encoder *enc, mist_av_packet *pkt)
{
	if (enc == NULL || enc->ctx == NULL || pkt == NULL) {
		return MIST_AV_ERR;
	}
	AVPacket *avpkt = av_packet_alloc();
	if (avpkt == NULL) {
		return MIST_AV_ERR;
	}
	int err = avcodec_receive_packet(enc->ctx, avpkt);
	if (err < 0) {
		av_packet_free(&avpkt);
		return map_ret(err);
	}
	err = copy_packet(avpkt, pkt);
	av_packet_free(&avpkt);
	return err < 0 ? MIST_AV_ERR : MIST_AV_OK;
}

void mist_av_encoder_close(mist_av_encoder *enc)
{
	if (enc == NULL) {
		return;
	}
	avcodec_free_context(&enc->ctx);
	av_free(enc);
}

void mist_av_packet_unref(mist_av_packet *pkt)
{
	if (pkt == NULL) {
		return;
	}
	av_free(pkt->data);
	pkt->data = NULL;
	pkt->size = 0;
}

void mist_av_frame_unref(mist_av_frame *frame)
{
	if (frame == NULL) {
		return;
	}
	for (int i = 0; i < 8; i++) {
		av_free(frame->data[i]);
		frame->data[i] = NULL;
	}
}
