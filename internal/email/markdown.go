// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package email

import (
	"regexp"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"golang.org/x/net/html"
)

var urlRegexp = regexp.MustCompile(`^(?:https?)://[-a-zA-Z0-9@:%._\+~#=]{1,256}(?::\d+)?(?:[/#?][-a-zA-Z0-9@:%_+.~#$!?&/=;,'">\^{}\[\]` + "`" + `]*)?`)

// markdown is the main markdown parser for email text to html.
var markdown = goldmark.New(
	goldmark.WithExtensions(
		extension.NewLinkify(
			extension.WithLinkifyAllowedProtocols([]string{
				"http:",
				"https:",
			}),
			extension.WithLinkifyURLRegexp(
				urlRegexp,
			),
		),
	),
)

var html2md4email = converter.NewConverter(
	converter.WithPlugins(
		base.NewBasePlugin(),
		commonmark.NewCommonmarkPlugin(
			commonmark.WithHeadingStyle(commonmark.HeadingStyleSetext),
			commonmark.WithEmDelimiter("_"),
			commonmark.WithStrongDelimiter("**"),
			commonmark.WithBulletListMarker("-"),
		),
		table.NewTablePlugin(),
		&html2mdMailPlugin{},
	),
)

type html2mdMailPlugin struct{}

func (s *html2mdMailPlugin) Name() string {
	return "email-render"
}

func (s *html2mdMailPlugin) Init(conv *converter.Converter) error {
	conv.Register.RendererFor("img", converter.TagTypeInline, s.ignoreImg, converter.PriorityEarly)
	return nil
}

func (s *html2mdMailPlugin) ignoreImg(_ converter.Context, _ converter.Writer, _ *html.Node) converter.RenderStatus {
	return converter.RenderSuccess
}
