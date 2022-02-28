package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

var (
	m3u = "https://raw.githubusercontent.com/qwerttvv/Beijing-IPTV/master/IPTV-Unicom.m3u"
	epg = "http://epg.51zmt.top:8000"

	regXTvgUrl, _ = regexp.Compile("x-tvg-url=\"(.+)\"")
	regTvgName, _ = regexp.Compile("tvg-name=\"(.+)\"")
)

type ExtM3u struct {
	XTvgUrl string
	ExtInf  []*ExtInf
}

type ExtInf struct {
	Name        string
	TvgId       string
	TvgCountry  string
	TvgLanguage string
	TvgLogo     string
	GroupTitle  string
	TvgName     string
	M3u         string
}

type Channel struct {
	Name     string
	TvgName  string
	Category string
	Source   string
	Logo     string
}

func main() {
	content := httpGet(epg)
	if content == "" {
		return
	}

	channel := parseTvLogo(content)
	if len(channel) == 0 {
		return
	}

	content = httpGet(m3u)
	if content == "" {
		return
	}

	em := parseM3u(content, channel)
	if em == nil {
		return
	}

	m3uString := buildM3u(em)
	err := ioutil.WriteFile("IPTV-Unicom.m3u", []byte(m3uString), 0664)
	if err != nil {
		panic(err)
	}
}

func buildM3u(em *ExtM3u) string {
	lines := make([]string, 0)

	lines = append(lines, fmt.Sprintf("#EXTM3U x-tvg-url=\"%s\"", em.XTvgUrl))

	for _, item := range em.ExtInf {
		tag := make([]string, 0)
		if item.TvgName != "" {
			tag = append(tag, fmt.Sprintf("tvg-name=\"%s\"", item.TvgName))
		}
		if item.TvgLogo != "" {
			tag = append(tag, fmt.Sprintf("tvg-logo=\"%s\"", item.TvgLogo))
		}
		if item.GroupTitle == "" {
			item.GroupTitle = "其他"
		}
		tag = append(tag, fmt.Sprintf("group-title=\"%s\"", item.GroupTitle))
		tagString := ""
		if len(tag) > 0 {
			tagString = strings.Join(tag, " ")
		}

		lines = append(lines, fmt.Sprintf("#EXTINF:-1 %s,%s", tagString, item.Name))
		lines = append(lines, strings.Replace(item.M3u, "192.168.123.1:23234", "192.168.50.1:4022", 1))
	}

	return strings.Join(lines, "\n")
}

func parseTvLogo(source string) map[string]*Channel {
	reg, err := regexp.Compile("(?U)<tr\\s*>\\s+<th>[0-9]+</th>[\\s\\S]+</tr>")
	if err != nil {
		panic(err)
	}

	find := reg.FindAllStringSubmatch(source, -1)
	if find == nil || len(find) == 0 {
		return nil
	}

	list := make(map[string]*Channel, 0)
	regTd, _ := regexp.Compile("(?U)<td>(.+)</td>")
	regImg, _ := regexp.Compile("<img src=\"(.+)\" alt=\".+\" height=\"[0-9]+\">")
	for _, item := range find {
		findTd := regTd.FindAllStringSubmatch(item[0], -1)
		if findTd == nil || len(findTd) == 0 {
			continue
		}

		row := &Channel{
			TvgName:  findTd[2][1],
			Category: findTd[4][1],
			Source:   findTd[5][1],
		}

		findImg := regImg.FindStringSubmatch(findTd[0][1])
		if findImg != nil && len(findImg) == 2 {
			row.Logo = findImg[1]
		}

		list[findTd[2][1]] = row
	}

	return list
}

func parseM3u(source string, channel map[string]*Channel) *ExtM3u {
	split := strings.Split(source, "\n")
	if split[0][0:7] != "#EXTM3U" {
		return nil
	}

	em := &ExtM3u{
		XTvgUrl: "",
		ExtInf:  make([]*ExtInf, 0),
	}

	// #EXTM3U x-tvg-url="http://epg.51zmt.top:8000/e.xml.gz"
	find := regXTvgUrl.FindStringSubmatch(split[0])
	if find != nil && len(find) == 2 {
		em.XTvgUrl = find[1]
	}

	for i := 1; i < len(split); i++ {
		ei := split[i]
		if len(ei) < 7 || ei[0:7] != "#EXTINF" {
			continue
		}

		//  注释掉的行： # http://hdtv.haust.edu.cn/hls/hls184a518.m3u8
		m3u := split[i+1]
		if m3u[0:4] != "http" {
			for j := i + 2; j < len(split); j++ {
				m3u = split[j]
				if m3u[0:4] == "http" || (len(m3u) >= 7 && m3u[0:7] == "#EXTINF") {
					i = j
					break
				}
			}

			if len(m3u) >= 7 && m3u[0:7] == "#EXTINF" {
				continue
			}
		}

		//#EXTINF:-1 tvg-name="CCTV5+",CCTV-5+ 体育赛事[高清]
		row := &ExtInf{
			M3u: strings.Trim(m3u, "\r\n"),
		}

		// tvg-name="CCTV5+"
		find := regTvgName.FindStringSubmatch(ei)
		if find != nil && len(find) == 2 {
			row.TvgName = find[1]
			if sub, ok := channel[row.TvgName]; ok {
				row.TvgLogo = sub.Logo
				row.GroupTitle = sub.Category
			}
		}

		// CCTV-5+ 体育赛事[高清]
		eiSplit := strings.Split(ei, ",")
		if len(eiSplit) == 2 {
			row.Name = strings.Trim(eiSplit[1], "\r\n")
		}

		em.ExtInf = append(em.ExtInf, row)
	}

	return em
}

func httpGet(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		panic(err)
	}

	return string(bytes)
}
