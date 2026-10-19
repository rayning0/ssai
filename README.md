# ssai
 A mini-SSAI service

## Fast local live-stream loop

Convert the two MP4 files into compatible MPEG-TS segments:

```bash
mkdir -p testdata/program testdata/ad
ffmpeg -y -i testdata/30-sec-music-sea.mp4 -vf "scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2" -r 30 -c:v libx264 -preset veryfast -pix_fmt yuv420p -g 60 -keyint_min 60 -sc_threshold 0 -c:a aac -ar 48000 -b:a 128k -force_key_frames "expr:gte(t,n_forced*6)" -f hls -hls_time 6 -hls_list_size 0 -hls_segment_type mpegts -hls_flags independent_segments -hls_segment_filename 'testdata/program/program%03d.ts' testdata/program/index.m3u8
ffmpeg -y -i testdata/10-sec-countdown-ad.mp4 -f lavfi -i anullsrc=channel_layout=stereo:sample_rate=48000 -vf "scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2" -r 30 -map 0:v:0 -map 1:a:0 -c:v libx264 -preset veryfast -pix_fmt yuv420p -g 60 -keyint_min 60 -sc_threshold 0 -c:a aac -ar 48000 -b:a 128k -shortest -force_key_frames "expr:gte(t,n_forced*6)" -f hls -hls_time 6 -hls_list_size 0 -hls_segment_type mpegts -hls_flags independent_segments -hls_segment_filename 'testdata/ad/ad%03d.ts' testdata/ad/index.m3u8
```

Run the local fake HLS origin in one terminal. It serves a sliding live
playlist that loops 30 seconds of program followed by 10 seconds of ad:

```bash
go run ./cmd/fake-origin
```

Then point the SSAI service at it from another terminal:

```bash
go run ./cmd/server -mode=live -source-url=http://localhost:8090/live/fake.m3u8
```

Open `http://localhost:8080/live/wcvb.m3u8` in VLC or ffplay. URLs such as
`/segments/17.ts` are virtual live-sequence URLs: the fake origin maps them to
the fixed files in `testdata/program/` and `testdata/ad/`; no renaming occurs.
