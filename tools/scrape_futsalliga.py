#!/usr/bin/env python3
"""Scrape FC Bizoni UH data from 2fl-v.futsalliga.cz into the club.json shape
the site frontend expects (same as facr.tdvorak.dev API output).

Usage: python tools/scrape_futsalliga.py [--out data/club.json]
"""
import json
import re
import sys
import urllib.request
from datetime import datetime, timezone

sys.stdout.reconfigure(encoding="utf-8", errors="replace")

BASE = "https://2fl-v.futsalliga.cz"
SEASON = "2027"           # 2026/2027
TEAM_ID = "60"            # FC Bizoni Uherske Hradiste
CLUB_NAME = "FC Bizoni Uherské Hradiště"
COMP_NAME = "2. Futsal liga - východ 2026/2027"
CLUB_LOGO = "/img/logo.png"

UA = {"User-Agent": "Mozilla/5.0 (bizoni-site-scraper)"}


def fetch(url):
    req = urllib.request.Request(url, headers=UA)
    with urllib.request.urlopen(req, timeout=30) as r:
        return r.read().decode("utf-8", "replace")


def strip_tags(s):
    return re.sub(r"<[^>]+>", "", s)


def clean(s):
    return re.sub(r"\s+", " ", strip_tags(s)).strip()


def team_logo(content):
    m = re.search(r'(?:data-src|src)="(https?://[^"]+)"', content)
    return m.group(1) if m else ""


def is_bizoni(name):
    return "bizoni" in name.lower()


def parse_matches(html_text):
    # Each date group: <h2 class="delta ...">...weekday D. M. YYYY</h2> followed by match rows
    h2re = re.compile(r'<h2 class="delta[^"]*"[^>]*>.*?</h2>', re.S)
    headers = list(h2re.finditer(html_text))
    matches = []
    for i, h in enumerate(headers):
        date_m = re.search(r"(\d{1,2})\.\s*(\d{1,2})\.\s*(\d{4})", strip_tags(h.group(0)))
        if not date_m:
            continue
        d, mo, y = int(date_m.group(1)), int(date_m.group(2)), int(date_m.group(3))
        section = html_text[h.end():headers[i + 1].start() if i + 1 < len(headers) else len(html_text)]
        # each match row starts with the time anchor
        rowre = re.compile(
            r'<a href="/match/(\d+)"[^>]*>\s*<div class="delta mb-0 pl-md-3 pl-lg-8">\s*([0-9:]+)\s*</div>',
            re.S,
        )
        rows = list(rowre.finditer(section))
        for j, row in enumerate(rows):
            seg = section[row.end():rows[j + 1].start() if j + 1 < len(rows) else len(section)]
            match_id, time_str = row.group(1), row.group(2)
            teams = re.findall(r'<div class="match-team">(.*?)</div>', seg, re.S)
            if len(teams) < 2:
                continue
            home, away = clean(teams[0]), clean(teams[1])
            home_logo, away_logo = team_logo(teams[0]), team_logo(teams[1])
            score = ""
            sm = re.search(r'match-results[^>]*>\s*([^<]*?)\s*</div>', seg, re.S)
            if sm:
                sc = re.search(r"(\d+)\s*:\s*(\d+)", sm.group(1))
                if sc:
                    score = f"{sc.group(1)}:{sc.group(2)}"
            vm = re.search(r'<div class="match-place">\s*([^<]*?)\s*</div>', seg, re.S)
            venue = vm.group(1).strip() if vm else ""
            if is_bizoni(home):
                home_logo = CLUB_LOGO
            if is_bizoni(away):
                away_logo = CLUB_LOGO
            matches.append({
                "date_time": f"{d:02d}.{mo:02d}.{y:04d} {time_str}",
                "home": home,
                "home_id": TEAM_ID if is_bizoni(home) else "",
                "home_logo_url": home_logo,
                "away": away,
                "away_id": TEAM_ID if is_bizoni(away) else "",
                "away_logo_url": away_logo,
                "score": score,
                "venue": venue,
                "match_id": match_id,
                "report_url": f"{BASE}/match/{match_id}",
                "facr_link": f"{BASE}/match/{match_id}",
            })
    return matches


def parse_standings(html_text):
    tm = re.search(r'<table[^>]*standingsTable[^>]*>(.*?)</table>', html_text, re.S)
    if not tm:
        return []
    rows = []
    for tr in re.findall(r"<tr>(.*?)</tr>", tm.group(1), re.S):
        cells = re.findall(r"<td[^>]*>(.*?)</td>", tr, re.S)
        if len(cells) < 8:
            continue
        team = clean(cells[1])
        rows.append({
            "rank": clean(cells[0]),
            "team": team,
            "team_id": TEAM_ID if is_bizoni(team) else "",
            "team_logo_url": CLUB_LOGO if is_bizoni(team) else team_logo(cells[1]),
            "played": clean(cells[2]),
            "wins": clean(cells[3]),
            "draws": clean(cells[4]),
            "losses": clean(cells[5]),
            "score": clean(cells[6]),
            "points": clean(cells[7]),
        })
    return rows


def main():
    out = sys.argv[sys.argv.index("--out") + 1] if "--out" in sys.argv else None
    matches_html = fetch(f"{BASE}/matches/{SEASON}?team={TEAM_ID}")
    standings_html = fetch(f"{BASE}/standings/{SEASON}")
    matches = parse_matches(matches_html)
    table = parse_standings(standings_html)
    comp = {
        "id": SEASON,
        "code": "2FL-V",
        "name": COMP_NAME,
        "team_count": str(len(table)),
        "matches_link": f"{BASE}/matches/{SEASON}?team={TEAM_ID}",
        "matches": matches,
    }
    detail = {
        "name": CLUB_NAME,
        "club_id": TEAM_ID,
        "club_type": "futsal",
        "url": f"{BASE}/team/{TEAM_ID}",
        "logo_url": CLUB_LOGO,
        "address": "",
        "category": "Futsal",
        "competitions": [comp],
    }
    table_obj = {
        "name": CLUB_NAME,
        "club_id": TEAM_ID,
        "club_type": "futsal",
        "logo_url": CLUB_LOGO,
        "competitions": [{
            "id": SEASON,
            "code": "2FL-V",
            "name": COMP_NAME,
            "team_count": str(len(table)),
            "matches_link": comp["matches_link"],
            "table": {"overall": table},
        }],
    }
    combined = {
        "fetched_at": datetime.now(timezone.utc).isoformat(),
        "club_detail": detail,
        "club_table": table_obj,
    }
    print(f"matches={len(matches)} standings_rows={len(table)}")
    for m in matches[:4]:
        print(f"  {m['date_time']}  {m['home']} {m['score'] or '-:-'} {m['away']}  @ {m['venue']}")
    print("  ...")
    for m in matches[-2:]:
        print(f"  {m['date_time']}  {m['home']} {m['score'] or '-:-'} {m['away']}  @ {m['venue']}")
    if out:
        with open(out, "w", encoding="utf-8") as f:
            json.dump(combined, f, ensure_ascii=False, indent=2)
        print(f"wrote {out}")


if __name__ == "__main__":
    main()
