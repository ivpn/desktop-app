
// The sources is a small copy of wireless-tools (iwlib.c)

#ifndef IWLIB_COPY
#define IWLIB_COPY

#include <iwlib.h> // sudo apt-get install libiw-dev

/* ---------------------------- TYPES ---------------------------- */

/*
 * Describe how a standard IOCTL looks like.
 */
struct iw_ioctl_description
{
	__u8	header_type;		/* NULL, iw_point or other */
	__u8	token_type;		/* Future */
	__u16	token_size;		/* Granularity of payload */
	__u16	min_tokens;		/* Min acceptable token number */
	__u16	max_tokens;		/* Max acceptable token number */
	__u32	flags;			/* Special handling of the request */
};
/* -------------------------- CONSTANTS -------------------------- */
/* Type of headers we know about (basically union iwreq_data) */
#define IW_HEADER_TYPE_NULL	0	/* Not available */
#define IW_HEADER_TYPE_CHAR	2	/* char [IFNAMSIZ] */
#define IW_HEADER_TYPE_UINT	4	/* __u32 */
#define IW_HEADER_TYPE_FREQ	5	/* struct iw_freq */
#define IW_HEADER_TYPE_ADDR	6	/* struct sockaddr */
#define IW_HEADER_TYPE_POINT	8	/* struct iw_point */
#define IW_HEADER_TYPE_PARAM	9	/* struct iw_param */
#define IW_HEADER_TYPE_QUAL	10	/* struct iw_quality */

/* Handling flags */
/* Most are not implemented. I just use them as a reminder of some
 * cool features we might need one day ;-) */
#define IW_DESCR_FLAG_NONE	0x0000	/* Obvious */
/* Wrapper level flags */
#define IW_DESCR_FLAG_DUMP	0x0001	/* Not part of the dump command */
#define IW_DESCR_FLAG_EVENT	0x0002	/* Generate an event on SET */
#define IW_DESCR_FLAG_RESTRICT	0x0004	/* GET : request is ROOT only */
				/* SET : Omit payload from generated iwevent */
#define IW_DESCR_FLAG_NOMAX	0x0008	/* GET : no limit on request size */
/* Driver level flags */
#define IW_DESCR_FLAG_WAIT	0x0100	/* Wait for driver event */

/* -------------------------- VARIABLES -------------------------- */
/*
 * Meta-data about all the standard Wireless Extension request we
 * know about.
 */
static const struct iw_ioctl_description standard_ioctl_descr[] = {
    
	[SIOCSIWCOMMIT	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_NULL,
	},
	[SIOCGIWNAME	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_CHAR,
		.flags		= IW_DESCR_FLAG_DUMP,
	},
	[SIOCSIWNWID	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
		.flags		= IW_DESCR_FLAG_EVENT,
	},
	[SIOCGIWNWID	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
		.flags		= IW_DESCR_FLAG_DUMP,
	},
	[SIOCSIWFREQ	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_FREQ,
		.flags		= IW_DESCR_FLAG_EVENT,
	},
	[SIOCGIWFREQ	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_FREQ,
		.flags		= IW_DESCR_FLAG_DUMP,
	},
	[SIOCSIWMODE	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_UINT,
		.flags		= IW_DESCR_FLAG_EVENT,
	},
	[SIOCGIWMODE	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_UINT,
		.flags		= IW_DESCR_FLAG_DUMP,
	},
	[SIOCSIWSENS	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCGIWSENS	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCSIWRANGE	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_NULL,
	},
	[SIOCGIWRANGE	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= sizeof(struct iw_range),
		.flags		= IW_DESCR_FLAG_DUMP,
	},
	[SIOCSIWPRIV	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_NULL,
	},
	[SIOCGIWPRIV	- SIOCIWFIRST] = { 
		.header_type	= IW_HEADER_TYPE_NULL,
	},
	[SIOCSIWSTATS	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_NULL,
	},
	[SIOCGIWSTATS	- SIOCIWFIRST] = { 
		.header_type	= IW_HEADER_TYPE_NULL,
		.flags		= IW_DESCR_FLAG_DUMP,
	},
	[SIOCSIWSPY	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= sizeof(struct sockaddr),
		.max_tokens	= IW_MAX_SPY,
	},
	[SIOCGIWSPY	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= sizeof(struct sockaddr) +
				  sizeof(struct iw_quality),
		.max_tokens	= IW_MAX_SPY,
	},
	[SIOCSIWTHRSPY	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= sizeof(struct iw_thrspy),
		.min_tokens	= 1,
		.max_tokens	= 1,
	},
	[SIOCGIWTHRSPY	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= sizeof(struct iw_thrspy),
		.min_tokens	= 1,
		.max_tokens	= 1,
	},
	[SIOCSIWAP	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_ADDR,
	},
	[SIOCGIWAP	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_ADDR,
		.flags		= IW_DESCR_FLAG_DUMP,
	},
	[SIOCSIWMLME	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.min_tokens	= sizeof(struct iw_mlme),
		.max_tokens	= sizeof(struct iw_mlme),
	},
	[SIOCGIWAPLIST	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= sizeof(struct sockaddr) +
				  sizeof(struct iw_quality),
		.max_tokens	= IW_MAX_AP,
		.flags		= IW_DESCR_FLAG_NOMAX,
	},
	[SIOCSIWSCAN	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.min_tokens	= 0,
		.max_tokens	= sizeof(struct iw_scan_req),
	},
	[SIOCGIWSCAN	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_SCAN_MAX_DATA,
		.flags		= IW_DESCR_FLAG_NOMAX,
	},
	[SIOCSIWESSID	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_ESSID_MAX_SIZE + 1,
		.flags		= IW_DESCR_FLAG_EVENT,
	},
	[SIOCGIWESSID	- SIOCIWFIRST] = { //!!!!!
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_ESSID_MAX_SIZE + 1,
		.flags		= IW_DESCR_FLAG_DUMP,
	},
	[SIOCSIWNICKN	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_ESSID_MAX_SIZE + 1,
	},
	[SIOCGIWNICKN	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_ESSID_MAX_SIZE + 1,
	},
    [SIOCGIWNICKN	- SIOCIWFIRST +1] = {
		.header_type	= IW_HEADER_TYPE_NULL,
	},
    [SIOCGIWNICKN	- SIOCIWFIRST +2] = {
		.header_type	= IW_HEADER_TYPE_NULL,
	},
	[SIOCSIWRATE	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCGIWRATE	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCSIWRTS	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCGIWRTS	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCSIWFRAG	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCGIWFRAG	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCSIWTXPOW	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCGIWTXPOW	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCSIWRETRY	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCGIWRETRY	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCSIWENCODE	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_ENCODING_TOKEN_MAX,
		.flags		= IW_DESCR_FLAG_EVENT | IW_DESCR_FLAG_RESTRICT,
	},
	[SIOCGIWENCODE	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_ENCODING_TOKEN_MAX,
		.flags		= IW_DESCR_FLAG_DUMP | IW_DESCR_FLAG_RESTRICT,
	},
	[SIOCSIWPOWER	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCGIWPOWER	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCSIWMODUL	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCGIWMODUL	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCSIWGENIE	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_GENERIC_IE_MAX,
	},
	[SIOCGIWGENIE	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_GENERIC_IE_MAX,
	},
	[SIOCSIWAUTH	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCGIWAUTH	- SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_PARAM,
	},
	[SIOCSIWENCODEEXT - SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.min_tokens	= sizeof(struct iw_encode_ext),
		.max_tokens	= sizeof(struct iw_encode_ext) +
				  IW_ENCODING_TOKEN_MAX,
	},
	[SIOCGIWENCODEEXT - SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.min_tokens	= sizeof(struct iw_encode_ext),
		.max_tokens	= sizeof(struct iw_encode_ext) +
				  IW_ENCODING_TOKEN_MAX,
	},
	[SIOCSIWPMKSA - SIOCIWFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.min_tokens	= sizeof(struct iw_pmksa),
		.max_tokens	= sizeof(struct iw_pmksa),
	},
};
static const unsigned int standard_ioctl_num = (sizeof(standard_ioctl_descr) /
						sizeof(struct iw_ioctl_description));

/*
 * Meta-data about all the additional standard Wireless Extension events
 * we know about.
 */
static const struct iw_ioctl_description standard_event_descr[] = {
	[IWEVTXDROP	- IWEVFIRST] = {
		.header_type	= IW_HEADER_TYPE_ADDR,
	},
	[IWEVQUAL	- IWEVFIRST] = {
		.header_type	= IW_HEADER_TYPE_QUAL,
	},
	[IWEVCUSTOM	- IWEVFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_CUSTOM_MAX,
	},
	[IWEVREGISTERED	- IWEVFIRST] = {
		.header_type	= IW_HEADER_TYPE_ADDR,
	},
	[IWEVEXPIRED	- IWEVFIRST] = {
		.header_type	= IW_HEADER_TYPE_ADDR, 
	},
	[IWEVGENIE	- IWEVFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_GENERIC_IE_MAX,
	},
	[IWEVMICHAELMICFAILURE	- IWEVFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT, 
		.token_size	= 1,
		.max_tokens	= sizeof(struct iw_michaelmicfailure),
	},
	[IWEVASSOCREQIE	- IWEVFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_GENERIC_IE_MAX,
	},
	[IWEVASSOCRESPIE	- IWEVFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= IW_GENERIC_IE_MAX,
	},
	[IWEVPMKIDCAND	- IWEVFIRST] = {
		.header_type	= IW_HEADER_TYPE_POINT,
		.token_size	= 1,
		.max_tokens	= sizeof(struct iw_pmkid_cand),
	},
};
static const unsigned int standard_event_num = (sizeof(standard_event_descr) /
						sizeof(struct iw_ioctl_description));


/* Size (in bytes) of various events */
static const int event_type_size[] = {
	IW_EV_LCP_PK_LEN,	/* IW_HEADER_TYPE_NULL */
	0,
	IW_EV_CHAR_PK_LEN,	/* IW_HEADER_TYPE_CHAR */
	0,
	IW_EV_UINT_PK_LEN,	/* IW_HEADER_TYPE_UINT */
	IW_EV_FREQ_PK_LEN,	/* IW_HEADER_TYPE_FREQ */
	IW_EV_ADDR_PK_LEN,	/* IW_HEADER_TYPE_ADDR */
	0,
	IW_EV_POINT_PK_LEN,	/* Without variable payload */
	IW_EV_PARAM_PK_LEN,	/* IW_HEADER_TYPE_PARAM */
	IW_EV_QUAL_PK_LEN,	/* IW_HEADER_TYPE_QUAL */
};  

/* Disable runtime version warning in iw_get_range_info() */
static int	iw_ignore_version = 0;

/*------------------------------------------------------------------*/
 /*
 * Get the range information out of the driver
 */
static inline int
iw_get_range_info_2(int		skfd,
		  const char *	ifname,
		  iwrange *	range)
{
  struct iwreq		wrq;
  char			buffer[sizeof(iwrange) * 2];	/* Large enough */

  /* Cleanup */
  memset(buffer, 0, sizeof(buffer));

  wrq.u.data.pointer = (caddr_t) buffer;
  wrq.u.data.length = sizeof(buffer);
  wrq.u.data.flags = 0;
  if(iw_get_ext(skfd, ifname, SIOCGIWRANGE, &wrq) < 0)
    return(-1);

  /* Copy stuff at the right place, ignore extra */
  memcpy((char *) range, buffer, sizeof(iwrange));

  /* Lots of people have driver and tools out of sync as far as Wireless
   * Extensions are concerned. It's because /usr/include/linux/wireless.h
   * and /usr/src/linux/include/linux/wireless.h are different.
   * We try to catch this stuff here... */
  if(!iw_ignore_version)
    {
      /* For new versions, we can check the version directly, for old versions
       * we use magic. 300 bytes is a also magic number, don't touch... */
      if((WIRELESS_EXT > 10) && (wrq.u.data.length >= 300))
	{
#if WIRELESS_EXT > 10
	  /* Version verification - for new versions */
	  if(range->we_version_compiled != WIRELESS_EXT)
	    {
	      fprintf(stderr, "Warning: Driver for device %s has been compiled with version %d\n", ifname, range->we_version_compiled);
	      fprintf(stderr, "of Wireless Extension, while this program is using version %d.\n", WIRELESS_EXT);
	      fprintf(stderr, "Some things may be broken...\n\n");
	    }
	  /* Driver version verification */
	  if(range->we_version_compiled < range->we_version_source)
	    {
	      fprintf(stderr, "Warning: Driver for device %s recommend version %d of Wireless Extension,\n", ifname, range->we_version_source);
	      fprintf(stderr, "but has been compiled with version %d, therefore some driver features\n", range->we_version_compiled);
	      fprintf(stderr, "may not be available...\n\n");
	    }
#endif /* WIRELESS_EXT > 10 */
	}
      else
	{
	  /* Version verification - for old versions */
	  if(wrq.u.data.length != sizeof(iwrange))
	    {
	      fprintf(stderr, "Warning: Driver for device %s has been compiled with an ancient version\n", ifname);
	      fprintf(stderr, "of Wireless Extension, while this program is using version %d.\n", WIRELESS_EXT);
	      fprintf(stderr, "Some things may be broken...\n\n");
	    }
	}
    }
  /* Don't complain twice.
   * In theory, the test apply to each individual driver, but usually
   * all drivers are compiled from the same kernel, and most often
   * problem is the system/glibc headers. */
  iw_ignore_version = 1;

  /* Note : we are only trying to catch compile difference, not source.
   * If the driver source has not been updated to the latest, it doesn't
   * matter because the new fields are set to zero */

  return(0);
}

/*
 * Initialise the struct stream_descr so that we can extract
 * individual events from the event stream.
 */
static inline void
iw_init_event_stream_2(struct stream_descr *	stream,	/* Stream of events */
		     char *			data,
		     int			len)
{
  /* Cleanup */
  memset((char *) stream, '\0', sizeof(struct stream_descr));

  /* Set things up */
  stream->current = data;
  stream->end = data + len;
}


/*------------------------------------------------------------------*/
/*
 * Extract the next event from the event stream.
 */
static inline  int
iw_extract_event_stream_2(struct stream_descr *	stream,	/* Stream of events */
			struct iw_event *	iwe,	/* Extracted event */
			int			we_version)
{
  const struct iw_ioctl_description *	descr = NULL;
  int		event_type = 0;
  unsigned int	event_len = 1;		/* Invalid */
  char *	pointer;
  /* Don't "optimise" the following variable, it will crash */
  unsigned	cmd_index;		/* *MUST* be unsigned */

  /* Check for end of stream */
  if((stream->current + IW_EV_LCP_PK_LEN) > stream->end)
    return(0);

#ifdef DEBUG
  printf("DBG - stream->current = %p, stream->value = %p, stream->end = %p\n",
	 stream->current, stream->value, stream->end);
#endif

  /* Extract the event header (to get the event id).
   * Note : the event may be unaligned, therefore copy... */
  memcpy((char *) iwe, stream->current, IW_EV_LCP_PK_LEN);

#ifdef DEBUG
  printf("DBG - iwe->cmd = 0x%X, iwe->len = %d\n",
	 iwe->cmd, iwe->len);
#endif

  /* Check invalid events */
  if(iwe->len <= IW_EV_LCP_PK_LEN)
    return(-1);

  /* Get the type and length of that event */
  if(iwe->cmd <= SIOCIWLAST)
    {
      cmd_index = iwe->cmd - SIOCIWFIRST;
      if(cmd_index < standard_ioctl_num)
	descr = &(standard_ioctl_descr[cmd_index]);
    }
  else
    {
      cmd_index = iwe->cmd - IWEVFIRST;
      if(cmd_index < standard_event_num)
	descr = &(standard_event_descr[cmd_index]);
    }
  if(descr != NULL)
    event_type = descr->header_type;
  /* Unknown events -> event_type=0 => IW_EV_LCP_PK_LEN */
  event_len = event_type_size[event_type];
  /* Fixup for earlier version of WE */
  if((we_version <= 18) && (event_type == IW_HEADER_TYPE_POINT))
    event_len += IW_EV_POINT_OFF;

  /* Check if we know about this event */
  if(event_len <= IW_EV_LCP_PK_LEN)
    {
      /* Skip to next event */
      stream->current += iwe->len;
      return(2);
    }
  event_len -= IW_EV_LCP_PK_LEN;

  /* Set pointer on data */
  if(stream->value != NULL)
    pointer = stream->value;			/* Next value in event */
  else
    pointer = stream->current + IW_EV_LCP_PK_LEN;	/* First value in event */

#ifdef DEBUG
  printf("DBG - event_type = %d, event_len = %d, pointer = %p\n",
	 event_type, event_len, pointer);
#endif

  /* Copy the rest of the event (at least, fixed part) */
  if((pointer + event_len) > stream->end)
    {
      /* Go to next event */
      stream->current += iwe->len;
      return(-2);
    }
  /* Fixup for WE-19 and later : pointer no longer in the stream */
  /* Beware of alignement. Dest has local alignement, not packed */
  if((we_version > 18) && (event_type == IW_HEADER_TYPE_POINT))
    memcpy((char *) iwe + IW_EV_LCP_LEN + IW_EV_POINT_OFF,
	   pointer, event_len);
  else
    memcpy((char *) iwe + IW_EV_LCP_LEN, pointer, event_len);

  /* Skip event in the stream */
  pointer += event_len;

  /* Special processing for iw_point events */
  if(event_type == IW_HEADER_TYPE_POINT)
    {
      /* Check the length of the payload */
      unsigned int	extra_len = iwe->len - (event_len + IW_EV_LCP_PK_LEN);
      if(extra_len > 0)
	{
	  /* Set pointer on variable part (warning : non aligned) */
	  iwe->u.data.pointer = pointer;

	  /* Check that we have a descriptor for the command */
	  if(descr == NULL)
	    /* Can't check payload -> unsafe... */
	    iwe->u.data.pointer = NULL;	/* Discard paylod */
	  else
	    {
	      /* Those checks are actually pretty hard to trigger,
	       * because of the checks done in the kernel... */

	      unsigned int	token_len = iwe->u.data.length * descr->token_size;

	      /* Ugly fixup for alignement issues.
	       * If the kernel is 64 bits and userspace 32 bits,
	       * we have an extra 4+4 bytes.
	       * Fixing that in the kernel would break 64 bits userspace. */
	      if((token_len != extra_len) && (extra_len >= 4))
		{
		  __u16		alt_dlen = *((__u16 *) pointer);
		  unsigned int	alt_token_len = alt_dlen * descr->token_size;
		  if((alt_token_len + 8) == extra_len)
		    {
#ifdef DEBUG
		      printf("DBG - alt_token_len = %d\n", alt_token_len);
#endif
		      /* Ok, let's redo everything */
		      pointer -= event_len;
		      pointer += 4;
		      /* Dest has local alignement, not packed */
		      memcpy((char *) iwe + IW_EV_LCP_LEN + IW_EV_POINT_OFF,
			     pointer, event_len);
		      pointer += event_len + 4;
		      iwe->u.data.pointer = pointer;
		      token_len = alt_token_len;
		    }
		}

	      /* Discard bogus events which advertise more tokens than
	       * what they carry... */
	      if(token_len > extra_len)
		iwe->u.data.pointer = NULL;	/* Discard paylod */
	      /* Check that the advertised token size is not going to
	       * produce buffer overflow to our caller... */
	      if((iwe->u.data.length > descr->max_tokens)
		 && !(descr->flags & IW_DESCR_FLAG_NOMAX))
		iwe->u.data.pointer = NULL;	/* Discard paylod */
	      /* Same for underflows... */
	      if(iwe->u.data.length < descr->min_tokens)
		iwe->u.data.pointer = NULL;	/* Discard paylod */
#ifdef DEBUG
	      printf("DBG - extra_len = %d, token_len = %d, token = %d, max = %d, min = %d\n",
		     extra_len, token_len, iwe->u.data.length, descr->max_tokens, descr->min_tokens);
#endif
	    }
	}
      else
	/* No data */
	iwe->u.data.pointer = NULL;

      /* Go to next event */
      stream->current += iwe->len;
    }
  else
    {
      /* Ugly fixup for alignement issues.
       * If the kernel is 64 bits and userspace 32 bits,
       * we have an extra 4 bytes.
       * Fixing that in the kernel would break 64 bits userspace. */
      if((stream->value == NULL)
	 && ((((iwe->len - IW_EV_LCP_PK_LEN) % event_len) == 4)
	     || ((iwe->len == 12) && ((event_type == IW_HEADER_TYPE_UINT) ||
				      (event_type == IW_HEADER_TYPE_QUAL))) ))
	{
#ifdef DEBUG
	  printf("DBG - alt iwe->len = %d\n", iwe->len - 4);
#endif
	  pointer -= event_len;
	  pointer += 4;
	  /* Beware of alignement. Dest has local alignement, not packed */
	  memcpy((char *) iwe + IW_EV_LCP_LEN, pointer, event_len);
	  pointer += event_len;
	}

      /* Is there more value in the event ? */
      if((pointer + event_len) <= (stream->current + iwe->len))
	/* Go to next value */
	stream->value = pointer;
      else
	{
	  /* Go to next event */
	  stream->value = NULL;
	  stream->current += iwe->len;
	}
    }
  return(1);
}
#endif