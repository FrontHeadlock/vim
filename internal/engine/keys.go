package engine

import (
	"strings"
	"unicode/utf8"
)

// parseKeys 는 "diw", "cw bye<esc>" 같은 입력 문자열을 Key 시퀀스로 변환한다.
// 특수키 토큰: <esc> <cr> <bs> <c-r>. par 산출(레벨의 Solution 길이 계산)과
// 테스트의 feedKeys() 양쪽에서 공유하는 프로덕션 파서.
//
// 특수키 토큰은 ASCII 뿐이라 바이트 오프셋 스캔으로 충분하지만, 그 외 문자는
// rune 단위로 디코드한다 — 바이트 인덱싱을 쓰면 멀티바이트 UTF-8 한 글자가
// 여러 개의 깨진 Key 로 쪼개진다(par = len(ParseKeys(Solution)) 오염, A4).
func ParseKeys(s string) []Key {
	var out []Key
	i := 0
	for i < len(s) {
		if s[i] == '<' {
			if j := strings.IndexByte(s[i:], '>'); j >= 0 {
				tok := s[i+1 : i+j]
				switch tok {
				case "esc", "cr", "bs", "c-r":
					out = append(out, SpecialKey(tok))
					i += j + 1
					continue
				}
			}
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		out = append(out, RuneKey(r))
		i += size
	}
	return out
}

// KeysString 은 ParseKeys 의 역변환이다 — Key 시퀀스를 사람이 읽고 다시
// 붙여넣을 수 있는 문자열로 만든다(refactor_2.md B4: 클리어 화면의 "내 풀이"
// 표시·복사). 반드시 왕복 가능해야 한다: ParseKeys(KeysString(keys)) 가
// keys 와 같아야 par/재생 경로가 어긋나지 않는다.
func KeysString(keys []Key) string {
	var b strings.Builder
	for _, k := range keys {
		if k.S != "" {
			b.WriteByte('<')
			b.WriteString(k.S)
			b.WriteByte('>')
			continue
		}
		b.WriteRune(k.R)
	}
	return b.String()
}
