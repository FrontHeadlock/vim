package enginetest

import (
	"testing"

	. "vimquest/internal/engine"
)

// TestKeysStringRoundTrip 은 KeysString(ParseKeys 의 역변환)이 특수키와
// 일반 문자를 섞어도 ParseKeys 와 왕복 가능한지 확인한다 — "내 풀이" 문자열이
// 다시 파싱 가능해야 par/재생 경로가 어긋나지 않는다.
func TestKeysStringRoundTrip(t *testing.T) {
	cases := []string{
		"wdwj.",
		"cwbar<esc>w.w.",
		"5G0",
		"한글<esc>",
	}
	for _, s := range cases {
		keys := ParseKeys(s)
		got := KeysString(keys)
		if got != s {
			t.Errorf("KeysString(ParseKeys(%q)) = %q, want %q", s, got, s)
		}
		// 왕복 후 다시 파싱해도 Key 시퀀스 자체가 같아야 한다.
		keys2 := ParseKeys(got)
		if len(keys) != len(keys2) {
			t.Fatalf("%q: 왕복 후 Key 개수 불일치: %d vs %d", s, len(keys), len(keys2))
		}
		for i := range keys {
			if keys[i] != keys2[i] {
				t.Errorf("%q: keys[%d]=%+v want %+v", s, i, keys2[i], keys[i])
			}
		}
	}
}
