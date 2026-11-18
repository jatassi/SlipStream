package cues

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
)

// srtOf is a two-cue SRT whose text is lines, long enough for chardet
// to score.
func srtOf(lines ...string) string {
	var b strings.Builder
	for i, l := range lines {
		fmt.Fprintf(&b, "%d\n00:00:0%d,000 --> 00:00:0%d,000\n%s\n\n", i+1, i+1, i+2, l)
	}
	return b.String()
}

// One sample per row of the table, in the page a release in that language
// most often comes in. Pages within a row that encode a text to the same
// bytes can't be told apart, and need not be.
func TestEncodings(t *testing.T) {
	cases := []struct {
		lang, page string
		enc        encoding.Encoding
		text       string
	}{
		{"en", "windows-1252", charmap.Windows1252, srtOf("Café “quoted” – naïve façade.", "Ne t’inquiète pas, c’était déjà fait.")},
		{"fr", "windows-1252", charmap.Windows1252, srtOf("Où est la bibliothèque ? Je ne sais pas, l’élève l’a déjà cherchée.", "Ça m’étonnerait beaucoup qu’il soit là.")},
		{"cs", "windows-1250", charmap.Windows1250, srtOf("Příliš žluťoučký kůň úpěl ďábelské ódy.", "Šťastný řidič čekal na ženu před školou.")},
		{"pl", "iso-8859-2", charmap.ISO8859_2, srtOf("Zażółć gęślą jaźń, powiedział łagodnie.", "Świetnie, że przyszedłeś tak późno.")},
		{"ru", "windows-1251", charmap.Windows1251, srtOf("Съешь же ещё этих мягких французских булок, да выпей чаю.", "Я не знаю, что ты хочешь от меня услышать.")},
		{"uk", "koi8-u", charmap.KOI8U, srtOf("Чуєш їх, доцю, га? Кумедна ж ти, прощайся без ґольфів!", "Я не знаю, що ти хочеш від мене почути.")},
		{"el", "windows-1253", charmap.Windows1253, srtOf("Ξεσκεπάζω την ψυχοφθόρα βδελυγμία.", "Δεν ξέρω τι θέλεις να ακούσεις από μένα.")},
		{"tr", "windows-1254", charmap.Windows1254, srtOf("Pijamalı hasta yağız şoföre çabucak güvendi.", "Ne duymak istediğini bilmiyorum, gerçekten.")},
		{"he", "windows-1255", charmap.Windows1255, srtOf("דג סקרן שט בים מאוכזב ולפתע מצא חברה.", "אני לא יודע מה אתה רוצה לשמוע ממני.")},
		{"ar", "windows-1256", charmap.Windows1256, srtOf("نص حكيم له سر قاطع وذو شأن عظيم مكتوب على ثوب أخضر.", "لا أعرف ماذا تريد أن تسمع مني.")},
		{"lv", "windows-1257", charmap.Windows1257, srtOf("Glāžšķūņa rūķīši dzērumā čiepj Baha koncertflīģeļu vākus.", "Es nezinu, ko tu gribi no manis dzirdēt.")},
		{"vi", "windows-1258", charmap.Windows1258, srtOf("Tôi không biê\u0301t anh muô\u0301n nghe gi\u0300 tư\u0300 tôi.", "Đư\u0300ng lo, mo\u0323i chuyê\u0323n se\u0303 ô\u0309n thôi.")},
		{"th", "windows-874", charmap.Windows874, srtOf("เป็นมนุษย์สุดประเสริฐเลิศคุณค่า", "ฉันไม่รู้ว่าคุณอยากได้ยินอะไรจากฉัน")},
		{"zh", "gbk", simplifiedchinese.GBK, srtOf("我不知道你想从我这里听到什么。这件事情我们明天再说吧。", "他们已经离开了，我们也应该走了，不要再等了。")},
		{"zh", "big5", traditionalchinese.Big5, srtOf("我不知道你想從我這裡聽到什麼。這件事情我們明天再說吧。", "他們已經離開了，我們也應該走了，不要再等了。")},
		{"ja", "shift-jis", japanese.ShiftJIS, srtOf("私はあなたが何を聞きたいのか分かりません。", "それは明日また話しましょう。もう行かなければ。")},
		{"ja", "euc-jp", japanese.EUCJP, srtOf("私はあなたが何を聞きたいのか分かりません。", "それは明日また話しましょう。もう行かなければ。")},
		{"ja", "iso-2022-jp", japanese.ISO2022JP, srtOf("私はあなたが何を聞きたいのか分かりません。", "それは明日また話しましょう。")},
		{"ko", "euc-kr", korean.EUCKR, srtOf("나는 당신이 나에게서 무엇을 듣고 싶은지 모르겠습니다.", "내일 다시 이야기합시다. 이제 가야 해요.")},
		{"en", "utf-16le", unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM), srtOf("No mark, little end.", "Second line.")},
		{"en", "utf-16be", unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), srtOf("No mark, big end.", "Second line.")},
		{"ru", "utf-16le", unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM), srtOf("С меткой порядка байтов.", "Вторая строка.")},
		{"ru", "utf-16be", unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM), srtOf("С меткой порядка байтов.", "Вторая строка.")},
		{"ru", "utf-8", unicode.UTF8BOM, srtOf("С меткой порядка байтов.", "Вторая строка.")},
		{"ru", "utf-8", unicode.UTF8, srtOf("Без метки.", "Вторая строка.")},
	}
	for _, c := range cases {
		t.Run(c.lang+"/"+c.page, func(t *testing.T) {
			b, err := c.enc.NewEncoder().String(c.text)
			if err != nil {
				t.Fatalf("encode the sample: %v", err)
			}
			got, page, err := Decode([]byte(b), lang(t, c.lang))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != c.text {
				t.Logf("decoded as %s", page)
				t.Errorf("decoded to\n%s\nwant\n%s", got, c.text)
			}
			d, err := Parse([]byte(b), lang(t, c.lang), 0, limit)
			if err != nil {
				t.Fatal(err)
			}
			out := d.Bytes()
			if !utf8.Valid(out) || bytes.HasPrefix(out, []byte("\xef\xbb\xbf")) || string(out) != c.text {
				t.Errorf("Parse wrote %q; want %q in UTF-8 without a BOM", out, c.text)
			}
		})
	}
}

func TestUndecodable(t *testing.T) {
	_, _, err := Decode([]byte{0x81, 0x20, 0xFF, 0xFF, 0x80}, lang(t, "ja"))
	if !errors.Is(err, errUndecodable) {
		t.Fatalf("%v; want errUndecodable", err)
	}
}
