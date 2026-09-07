package drift

import (
	"testing"
	"time"
)

const youtubeFeedFixture = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns:yt="http://www.youtube.com/xml/schemas/2015" xmlns="http://www.w3.org/2005/Atom">
 <entry>
  <title>High Stakes - Formula DRIFT Las Vegas Teaser | Sep 24-26</title>
  <yt:videoId>2H1Vf9xbmDw</yt:videoId>
  <published>2026-08-23T18:30:00+00:00</published>
 </entry>
 <entry>
  <title>Formula DRIFT Seattle 2026 - PRO, Round 6 - Qualifying</title>
  <yt:videoId>zB6lkWi41kc</yt:videoId>
  <published>2026-08-22T05:06:04+00:00</published>
 </entry>
</feed>`

func TestParseYouTubeFeed(t *testing.T) {
	videos, err := parseYouTubeFeed([]byte(youtubeFeedFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(videos) != 2 {
		t.Fatalf("videos %d", len(videos))
	}
	if videos[0].ID != "2H1Vf9xbmDw" {
		t.Fatalf("video id %q", videos[0].ID)
	}
}

func TestMatchYouTubeVideo(t *testing.T) {
	videos, err := parseYouTubeFeed([]byte(youtubeFeedFixture))
	if err != nil {
		t.Fatal(err)
	}
	lv := Event{
		Name: "HIGH STAKES", Slug: "las-vegas", Hashtag: "FDLV", Venue: "Las Vegas Motor Speedway",
		Start: time.Date(2026, 9, 24, 14, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC),
	}
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	got := matchYouTubeVideo(lv, videos, now)
	want := "https://www.youtube.com/watch?v=2H1Vf9xbmDw"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestParseYouTubeWatchURL(t *testing.T) {
	html := `<a href="https://www.youtube.com/watch?v=OVm7aM2bC34">Watch</a>`
	got := parseYouTubeWatchURL(html)
	if got != "https://www.youtube.com/watch?v=OVm7aM2bC34" {
		t.Fatalf("got %q", got)
	}
}

func TestPickYouTubeURLLiveWindow(t *testing.T) {
	e := Event{
		Start: time.Date(2026, 9, 24, 14, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC),
	}
	now := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	got := pickYouTubeURL(e, "", "", now)
	if got != formulaDriftYouTubeLiveURL {
		t.Fatalf("got %q", got)
	}
}
