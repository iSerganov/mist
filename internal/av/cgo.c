#include "cgo.h"

#include <string.h>

static void set_err(char *errbuf, int errlen, const char *msg)
{
	if (errbuf == NULL || errlen <= 0) {
		return;
	}
	strncpy(errbuf, msg, (size_t)errlen - 1);
	errbuf[errlen - 1] = '\0';
}

int mist_av_init(void)
{
	return MIST_AV_UNIMPLEMENTED;
}

mist_av_demuxer *mist_av_demuxer_open(const char *url, char *errbuf, int errlen)
{
	(void)url;
	set_err(errbuf, errlen, "unimplemented");
	return NULL;
}

mist_av_demuxer *mist_av_demuxer_open_io(mist_av_io *io, char *errbuf, int errlen)
{
	(void)io;
	set_err(errbuf, errlen, "unimplemented");
	return NULL;
}

int mist_av_demuxer_audio_info(mist_av_demuxer *d, mist_av_audio_info *info)
{
	(void)d;
	(void)info;
	return MIST_AV_UNIMPLEMENTED;
}

int mist_av_demuxer_read(mist_av_demuxer *d, mist_av_packet *pkt)
{
	(void)d;
	(void)pkt;
	return MIST_AV_UNIMPLEMENTED;
}

void mist_av_demuxer_close(mist_av_demuxer *d)
{
	(void)d;
}

mist_av_muxer *mist_av_muxer_open(const char *url, const mist_av_audio_info *info, char *errbuf, int errlen)
{
	(void)url;
	(void)info;
	set_err(errbuf, errlen, "unimplemented");
	return NULL;
}

mist_av_muxer *mist_av_muxer_open_io(mist_av_io *io, const mist_av_audio_info *info, char *errbuf, int errlen)
{
	(void)io;
	(void)info;
	set_err(errbuf, errlen, "unimplemented");
	return NULL;
}

int mist_av_muxer_write_header(mist_av_muxer *m)
{
	(void)m;
	return MIST_AV_UNIMPLEMENTED;
}

int mist_av_muxer_write(mist_av_muxer *m, const mist_av_packet *pkt)
{
	(void)m;
	(void)pkt;
	return MIST_AV_UNIMPLEMENTED;
}

int mist_av_muxer_write_trailer(mist_av_muxer *m)
{
	(void)m;
	return MIST_AV_UNIMPLEMENTED;
}

void mist_av_muxer_close(mist_av_muxer *m)
{
	(void)m;
}

mist_av_decoder *mist_av_decoder_open(const mist_av_audio_info *info, char *errbuf, int errlen)
{
	(void)info;
	set_err(errbuf, errlen, "unimplemented");
	return NULL;
}

int mist_av_decoder_send(mist_av_decoder *dec, const mist_av_packet *pkt)
{
	(void)dec;
	(void)pkt;
	return MIST_AV_UNIMPLEMENTED;
}

int mist_av_decoder_receive(mist_av_decoder *dec, mist_av_frame *frame)
{
	(void)dec;
	(void)frame;
	return MIST_AV_UNIMPLEMENTED;
}

void mist_av_decoder_close(mist_av_decoder *dec)
{
	(void)dec;
}

mist_av_encoder *mist_av_encoder_open(const mist_av_audio_info *info, char *errbuf, int errlen)
{
	(void)info;
	set_err(errbuf, errlen, "unimplemented");
	return NULL;
}

int mist_av_encoder_send(mist_av_encoder *enc, const mist_av_frame *frame)
{
	(void)enc;
	(void)frame;
	return MIST_AV_UNIMPLEMENTED;
}

int mist_av_encoder_receive(mist_av_encoder *enc, mist_av_packet *pkt)
{
	(void)enc;
	(void)pkt;
	return MIST_AV_UNIMPLEMENTED;
}

void mist_av_encoder_close(mist_av_encoder *enc)
{
	(void)enc;
}

mist_av_io *mist_av_io_new(void *opaque, mist_av_read_fn read, mist_av_write_fn write, mist_av_seek_fn seek)
{
	(void)opaque;
	(void)read;
	(void)write;
	(void)seek;
	return NULL;
}

void mist_av_io_free(mist_av_io *io)
{
	(void)io;
}

void mist_av_packet_unref(mist_av_packet *pkt)
{
	(void)pkt;
}

void mist_av_frame_unref(mist_av_frame *frame)
{
	(void)frame;
}
