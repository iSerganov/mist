/*
 * Real libavformat / libavcodec / libavutil implementation.
 * Custom AVIO uses integer handles owned by Go (see io.go); C never stores
 * a Go pointer. Packet and frame payloads are copied with av_malloc so the
 * Go side can C.GoBytes and then mist_av_*_unref. Codec IDs stay Mist-local.
 */
#include "cgo.h"

#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavformat/version.h>
#include <libavutil/avutil.h>
#include <libavutil/channel_layout.h>
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

static int fill_info_from_par(const AVCodecParameters *par, int64_t duration_us, mist_av_audio_info *info)
{
	memset(info, 0, sizeof(*info));
	info->codec_id = from_av_codec(par->codec_id);
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
	par->codec_id = to_av_codec(info->codec_id);
	par->sample_rate = info->sample_rate;
	par->format = info->sample_fmt;
	par->bit_rate = info->bitrate;
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

int mist_av_init(void)
{
	avformat_network_init();
	av_log_set_level(AV_LOG_ERROR);
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
	int err = avformat_alloc_output_context2(&m->fmt, NULL, "ogg", url);
	if (err < 0 || m->fmt == NULL) {
		set_averr(errbuf, errlen, err, "alloc ogg");
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
	const AVCodec *codec = avcodec_find_decoder(to_av_codec(info->codec_id));
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

mist_av_encoder *mist_av_encoder_open(const mist_av_audio_info *info, char *errbuf, int errlen)
{
	if (info == NULL) {
		set_err(errbuf, errlen, "nil info");
		return NULL;
	}
	const AVCodec *codec = avcodec_find_encoder(to_av_codec(info->codec_id));
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
	e->ctx->sample_fmt = AV_SAMPLE_FMT_FLTP;
	e->ctx->bit_rate = info->bitrate > 0 ? info->bitrate : 64000;
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
	int copy = nplanes < ch ? nplanes : ch;
	if (enc->ctx->sample_fmt == AV_SAMPLE_FMT_FLTP) {
		for (int i = 0; i < copy; i++) {
			memcpy(fr->data[i], planes[i], (size_t)nb_samples * sizeof(float));
		}
	} else {
		av_frame_free(&fr);
		return MIST_AV_ERR;
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
