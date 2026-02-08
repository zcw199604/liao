package app

import "testing"

func TestSelectDouyinLivePhotoPair(t *testing.T) {
	downloads := []string{
		"https://example.com/a.jpg",
		"https://example.com/aweme/v1/play/?video_id=1",
	}

	img, vid, errMsg := selectDouyinLivePhotoPair(downloads, nil, nil)
	if errMsg != "" {
		t.Fatalf("err=%q", errMsg)
	}
	if img != 0 || vid != 1 {
		t.Fatalf("img=%d vid=%d", img, vid)
	}

	imageIndex := 0
	videoIndex := 1
	img, vid, errMsg = selectDouyinLivePhotoPair(downloads, &imageIndex, &videoIndex)
	if errMsg != "" {
		t.Fatalf("err=%q", errMsg)
	}
	if img != 0 || vid != 1 {
		t.Fatalf("img=%d vid=%d", img, vid)
	}

	badImage := 1
	img, vid, errMsg = selectDouyinLivePhotoPair(downloads, &badImage, &videoIndex)
	if errMsg == "" {
		t.Fatalf("expected error, got img=%d vid=%d", img, vid)
	}
}

func TestSelectDouyinLivePhotoPair_PartialIndexes(t *testing.T) {
	downloads := []string{
		"https://example.com/cover.jpg",
		"https://example.com/img2.jpg",
		"https://example.com/aweme/v1/play/?video_id=1",
	}

	imageIndex := 1
	img, vid, errMsg := selectDouyinLivePhotoPair(downloads, &imageIndex, nil)
	if errMsg != "" {
		t.Fatalf("err=%q", errMsg)
	}
	if img != 1 || vid != 2 {
		t.Fatalf("img=%d vid=%d", img, vid)
	}

	videoIndex := 2
	img, vid, errMsg = selectDouyinLivePhotoPair(downloads, nil, &videoIndex)
	if errMsg != "" {
		t.Fatalf("err=%q", errMsg)
	}
	if img != 0 || vid != 2 {
		t.Fatalf("img=%d vid=%d", img, vid)
	}
}

func TestSelectDouyinLivePhotoPair_GroupedLiveResourcesUseRankPairing(t *testing.T) {
	downloads := []string{
		"https://example.com/img1.jpg",
		"https://example.com/img2.jpg",
		"https://example.com/aweme/v1/play/?video_id=1",
		"https://example.com/aweme/v1/play/?video_id=2",
	}

	image0 := 0
	img, vid, errMsg := selectDouyinLivePhotoPair(downloads, &image0, nil)
	if errMsg != "" {
		t.Fatalf("err=%q", errMsg)
	}
	if img != 0 || vid != 2 {
		t.Fatalf("image0 should pair video2, got img=%d vid=%d", img, vid)
	}

	image1 := 1
	img, vid, errMsg = selectDouyinLivePhotoPair(downloads, &image1, nil)
	if errMsg != "" {
		t.Fatalf("err=%q", errMsg)
	}
	if img != 1 || vid != 3 {
		t.Fatalf("image1 should pair video3, got img=%d vid=%d", img, vid)
	}

	video2 := 2
	img, vid, errMsg = selectDouyinLivePhotoPair(downloads, nil, &video2)
	if errMsg != "" {
		t.Fatalf("err=%q", errMsg)
	}
	if img != 0 || vid != 2 {
		t.Fatalf("video2 should pair image0, got img=%d vid=%d", img, vid)
	}

	video3 := 3
	img, vid, errMsg = selectDouyinLivePhotoPair(downloads, nil, &video3)
	if errMsg != "" {
		t.Fatalf("err=%q", errMsg)
	}
	if img != 1 || vid != 3 {
		t.Fatalf("video3 should pair image1, got img=%d vid=%d", img, vid)
	}
}
