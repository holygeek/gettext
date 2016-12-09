// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2015-2016 Canonical Ltd
 *
 * Permission is hereby granted, free of charge, to any person
 * obtaining a copy of this software and associated documentation
 * files (the "Software"), to deal in the Software without
 * restriction, including without limitation the rights to use, copy,
 * modify, merge, publish, distribute, sublicense, and/or sell copies
 * of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be
 * included in all copies or substantial portions of the Software.

 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
 * NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
 * BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
 * ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up check.v1 into the "go test" runner
func Test(t *testing.T) { TestingT(t) }

type xgettextTestSuite struct {
}

var _ = Suite(&xgettextTestSuite{})

// test helper
func makeGoSourceFile(c *C, content []byte) string {
	fname := filepath.Join(c.MkDir(), "foo.go")
	err := ioutil.WriteFile(fname, []byte(content), 0644)
	c.Assert(err, IsNil)

	return fname
}

func (s *xgettextTestSuite) SetUpTest(c *C) {
	// our test defaults
	opts.NoLocation = false
	opts.AddCommentsTag = "TRANSLATORS:"
	opts.Keyword = "i18n.G"
	opts.KeywordPlural = "i18n.NG"
	opts.SortOutput = true
	opts.PackageName = "snappy"
	opts.MsgIDBugsAddress = "snappy-devel@lists.ubuntu.com"

	// mock time
	formatTime = func() string {
		return "2015-06-30 14:48+0200"
	}
}

func (s *xgettextTestSuite) TestFormatComment(c *C) {
	var tests = []struct {
		in  string
		out string
	}{
		{in: "// foo ", out: "#. foo\n"},
		{in: "/* foo */", out: "#. foo\n"},
		{in: "/* foo\n */", out: "#. foo\n"},
		{in: "/* foo\nbar   */", out: "#. foo\n#. bar\n"},
	}

	for _, test := range tests {
		c.Assert(formatComment(test.in), Equals, test.out)
	}
}

func (s *xgettextTestSuite) TestProcessFilesSimple(c *C) {
	fname := makeGoSourceFile(c, []byte(`package main

func main() {
    // TRANSLATORS: foo comment
    i18n.G("foo")
}
`))
	err := processFiles([]string{fname})
	c.Assert(err, IsNil)

	c.Assert(msgIDs, DeepEquals, map[string][]msgID{
		"foo": []msgID{
			{
				comment: "#. TRANSLATORS: foo comment\n",
				fname:   fname,
				line:    5,
			},
		},
	})
}

func (s *xgettextTestSuite) TestProcessFilesMultiple(c *C) {
	fname := makeGoSourceFile(c, []byte(`package main

func main() {
    // TRANSLATORS: foo comment
    i18n.G("foo")

    // TRANSLATORS: bar comment
    i18n.G("foo")
}
`))
	err := processFiles([]string{fname})
	c.Assert(err, IsNil)

	c.Assert(msgIDs, DeepEquals, map[string][]msgID{
		"foo": []msgID{
			{
				comment: "#. TRANSLATORS: foo comment\n",
				fname:   fname,
				line:    5,
			},
			{
				comment: "#. TRANSLATORS: bar comment\n",
				fname:   fname,
				line:    8,
			},
		},
	})
}

const header = `# SOME DESCRIPTIVE TITLE.
# Copyright (C) YEAR THE PACKAGE'S COPYRIGHT HOLDER
# This file is distributed under the same license as the PACKAGE package.
# FIRST AUTHOR <EMAIL@ADDRESS>, YEAR.
#
#, fuzzy
msgid   ""
msgstr  "Project-Id-Version: snappy\n"
        "Report-Msgid-Bugs-To: snappy-devel@lists.ubuntu.com\n"
        "POT-Creation-Date: 2015-06-30 14:48+0200\n"
        "PO-Revision-Date: YEAR-MO-DA HO:MI+ZONE\n"
        "Last-Translator: FULL NAME <EMAIL@ADDRESS>\n"
        "Language-Team: LANGUAGE <LL@li.org>\n"
        "Language: \n"
        "MIME-Version: 1.0\n"
        "Content-Type: text/plain; charset=CHARSET\n"
        "Content-Transfer-Encoding: 8bit\n"
`

func (s *xgettextTestSuite) TestWriteOutputSimple(c *C) {
	msgIDs = map[string][]msgID{
		"foo": []msgID{
			{
				fname:   "fname",
				line:    2,
				comment: "#. foo\n",
			},
		},
	}
	out := bytes.NewBuffer([]byte(""))
	writePotFile(out)

	expected := fmt.Sprintf(`%s
#. foo
#: fname:2
msgid   "foo"
msgstr  ""

`, header)
	c.Assert(out.String(), Equals, expected)
}

func (s *xgettextTestSuite) TestWriteOutputMultiple(c *C) {
	msgIDs = map[string][]msgID{
		"foo": []msgID{
			{
				fname:   "fname",
				line:    2,
				comment: "#. comment1\n",
			},
			{
				fname:   "fname",
				line:    4,
				comment: "#. comment2\n",
			},
		},
	}
	out := bytes.NewBuffer([]byte(""))
	writePotFile(out)

	expected := fmt.Sprintf(`%s
#. comment1
#. comment2
#: fname:2 fname:4
msgid   "foo"
msgstr  ""

`, header)
	c.Assert(out.String(), Equals, expected)
}

func (s *xgettextTestSuite) TestWriteOutputNoComment(c *C) {
	msgIDs = map[string][]msgID{
		"foo": []msgID{
			{
				fname: "fname",
				line:  2,
			},
		},
	}
	out := bytes.NewBuffer([]byte(""))
	writePotFile(out)

	expected := fmt.Sprintf(`%s
#: fname:2
msgid   "foo"
msgstr  ""

`, header)
	c.Assert(out.String(), Equals, expected)
}

func (s *xgettextTestSuite) TestWriteOutputNoLocation(c *C) {
	msgIDs = map[string][]msgID{
		"foo": []msgID{
			{
				fname: "fname",
				line:  2,
			},
		},
	}

	opts.NoLocation = true
	out := bytes.NewBuffer([]byte(""))
	writePotFile(out)

	expected := fmt.Sprintf(`%s
msgid   "foo"
msgstr  ""

`, header)
	c.Assert(out.String(), Equals, expected)
}

func (s *xgettextTestSuite) TestWriteOutputFormatHint(c *C) {
	msgIDs = map[string][]msgID{
		"foo": []msgID{
			{
				fname:      "fname",
				line:       2,
				formatHint: "c-format",
			},
		},
	}

	out := bytes.NewBuffer([]byte(""))
	writePotFile(out)

	expected := fmt.Sprintf(`%s
#: fname:2
#, c-format
msgid   "foo"
msgstr  ""

`, header)
	c.Assert(out.String(), Equals, expected)
}

func (s *xgettextTestSuite) TestWriteOutputPlural(c *C) {
	msgIDs = map[string][]msgID{
		"foo": []msgID{
			{
				msgidPlural: "plural",
				fname:       "fname",
				line:        2,
			},
		},
	}

	out := bytes.NewBuffer([]byte(""))
	writePotFile(out)

	expected := fmt.Sprintf(`%s
#: fname:2
msgid   "foo"
msgid_plural   "plural"
msgstr[0]  ""
msgstr[1]  ""

`, header)
	c.Assert(out.String(), Equals, expected)
}

func (s *xgettextTestSuite) TestWriteOutputSorted(c *C) {
	msgIDs = map[string][]msgID{
		"aaa": []msgID{
			{
				fname: "fname",
				line:  2,
			},
		},
		"zzz": []msgID{
			{
				fname: "fname",
				line:  2,
			},
		},
	}

	opts.SortOutput = true
	// we need to run this a bunch of times as the ordering might
	// be right by pure chance
	for i := 0; i < 10; i++ {
		out := bytes.NewBuffer([]byte(""))
		writePotFile(out)

		expected := fmt.Sprintf(`%s
#: fname:2
msgid   "aaa"
msgstr  ""

#: fname:2
msgid   "zzz"
msgstr  ""

`, header)
		c.Assert(out.String(), Equals, expected)
	}
}

func (s *xgettextTestSuite) TestIntegration(c *C) {
	fname := makeGoSourceFile(c, []byte(`package main

func main() {
    // TRANSLATORS: foo comment
    //              with multiple lines
    i18n.G("foo")

    // this comment has no translators tag
    i18n.G("abc")

    // TRANSLATORS: plural
    i18n.NG("singular", "plural", 99)

    i18n.G("zz %s")
}
`))

	// a real integration test :)
	outName := filepath.Join(c.MkDir(), "snappy.pot")
	os.Args = []string{"test-binary",
		"--output", outName,
		"--keyword", "i18n.G",
		"--keyword-plural", "i18n.NG",
		"--msgid-bugs-address", "snappy-devel@lists.ubuntu.com",
		"--package-name", "snappy",
		fname,
	}
	main()

	// verify its what we expect
	got, err := ioutil.ReadFile(outName)
	c.Assert(err, IsNil)
	expected := fmt.Sprintf(`%s
#: %[2]s:9
msgid   "abc"
msgstr  ""

#. TRANSLATORS: foo comment
#. with multiple lines
#: %[2]s:6
msgid   "foo"
msgstr  ""

#. TRANSLATORS: plural
#: %[2]s:12
msgid   "singular"
msgid_plural   "plural"
msgstr[0]  ""
msgstr[1]  ""

#: %[2]s:14
#, c-format
msgid   "zz %%s"
msgstr  ""

`, header, fname)
	c.Assert(string(got), Equals, expected)
}

func (s *xgettextTestSuite) TestIntegrationMultipleKeywords(c *C) {
	fname := makeGoSourceFile(c, []byte(`package main

func main() {
    // TRANSLATORS: foo comment
    //              with multiple lines
    i18n.G("foo")
    i18n.Translate("goo foo")

    // this comment has no translators tag
    i18n.G("abc")
    i18n.Translate("goo abc")

    // TRANSLATORS: plural
    i18n.NG("singular", "plural", 99)
    i18n.TranslatePlural("one", "many", 3)

    i18n.G("zz %s")
    i18n.Translate("yy %s")
}
`))

	outName := filepath.Join(c.MkDir(), "snappy.pot")
	os.Args = []string{"test-binary",
		"--output", outName,
		"--keyword", "i18n.G,i18n.Translate",
		"--keyword-plural", "i18n.NG,i18n.TranslatePlural",
		"--msgid-bugs-address", "snappy-devel@lists.ubuntu.com",
		"--package-name", "snappy",
		fname,
	}
	main()

	got, err := ioutil.ReadFile(outName)
	c.Assert(err, IsNil)
	expected := fmt.Sprintf(`%s
#: %[2]s:10
msgid   "abc"
msgstr  ""

#. TRANSLATORS: foo comment
#. with multiple lines
#: %[2]s:6
msgid   "foo"
msgstr  ""

#: %[2]s:11
msgid   "goo abc"
msgstr  ""

#: %[2]s:7
msgid   "goo foo"
msgstr  ""

#: %[2]s:15
msgid   "one"
msgid_plural   "many"
msgstr[0]  ""
msgstr[1]  ""

#. TRANSLATORS: plural
#: %[2]s:14
msgid   "singular"
msgid_plural   "plural"
msgstr[0]  ""
msgstr[1]  ""

#: %[2]s:18
#, c-format
msgid   "yy %%s"
msgstr  ""

#: %[2]s:17
#, c-format
msgid   "zz %%s"
msgstr  ""

`, header, fname)
	c.Assert(string(got), Equals, expected)
}

func (s *xgettextTestSuite) TestProcessFilesConcat(c *C) {
	fname := makeGoSourceFile(c, []byte(`package main

func main() {
    // TRANSLATORS: foo comment
    i18n.G("foo\n" + "bar\n" + "baz")
}
`))
	err := processFiles([]string{fname})
	c.Assert(err, IsNil)

	c.Assert(msgIDs, DeepEquals, map[string][]msgID{
		"foo\\nbar\\nbaz": []msgID{
			{
				comment: "#. TRANSLATORS: foo comment\n",
				fname:   fname,
				line:    5,
			},
		},
	})
}

func (s *xgettextTestSuite) TestProcessFilesWithQuote(c *C) {
	fname := makeGoSourceFile(c, []byte(fmt.Sprintf(`package main

func main() {
    i18n.G(%[1]s foo "bar"%[1]s)
}
`, "`")))
	err := processFiles([]string{fname})
	c.Assert(err, IsNil)

	out := bytes.NewBuffer([]byte(""))
	writePotFile(out)

	expected := fmt.Sprintf(`%s
#: %[2]s:4
msgid   " foo \"bar\""
msgstr  ""

`, header, fname)
	c.Check(out.String(), Equals, expected)

}

func (s *xgettextTestSuite) TestWriteOutputMultilines(c *C) {
	msgIDs = map[string][]msgID{
		"foo\\nbar\\nbaz": []msgID{
			{
				fname:   "fname",
				line:    2,
				comment: "#. foo\n",
			},
		},
	}
	out := bytes.NewBuffer([]byte(""))
	writePotFile(out)
	expected := fmt.Sprintf(`%s
#. foo
#: fname:2
msgid   "foo\n"
        "bar\n"
        "baz"
msgstr  ""

`, header)
	c.Assert(out.String(), Equals, expected)
}

func (s *xgettextTestSuite) TestWriteOutputTidy(c *C) {
	msgIDs = map[string][]msgID{
		"foo\\nbar\\nbaz": []msgID{
			{
				fname: "fname",
				line:  2,
			},
		},
		"zzz\\n": []msgID{
			{
				fname: "fname",
				line:  4,
			},
		},
	}
	out := bytes.NewBuffer([]byte(""))
	writePotFile(out)
	expected := fmt.Sprintf(`%s
#: fname:2
msgid   "foo\n"
        "bar\n"
        "baz"
msgstr  ""

#: fname:4
msgid   "zzz\n"
msgstr  ""

`, header)
	c.Assert(out.String(), Equals, expected)
}

func (s *xgettextTestSuite) TestProcessFilesWithDoubleQuote(c *C) {
	fname := makeGoSourceFile(c, []byte(`package main

func main() {
    i18n.G("foo \"bar\"")
}
`))
	err := processFiles([]string{fname})
	c.Assert(err, IsNil)

	out := bytes.NewBuffer([]byte(""))
	writePotFile(out)

	expected := fmt.Sprintf(`%s
#: %[2]s:4
msgid   "foo \"bar\""
msgstr  ""

`, header, fname)
	c.Check(out.String(), Equals, expected)

}
