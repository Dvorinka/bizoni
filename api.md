Search Clubs
GET /club/search?q=QUERY

Find clubs on fotbal.cz. Supports football and futsal clubs.

Example: https://facr.tdvorak.dev/club/search?q=Sparta

Response shape
{
  "query": "Sparta",
  "count": 2,
  "results": [
    {
      "name": "AC Sparta Praha",
      "club_id": "",
      "club_type": "football",
      "url": "https://www.fotbal.cz/...",
      "logo_url": "https://.../logo.png",
      "category": "Muži",
      "address": "..."
    }
  ]
}

Club Info + Matches
GET /club/{type}/{id} - id must be provided in the setup of the club

{type}: football
{id}: club UUID from fotbal.cz
Example: https://facr.tdvorak.dev/club/football/00000000-0000-0000-0000-000000000000

Response shape
{
  "name": "AC Sparta Praha",
  "club_id": "00000000-0000-0000-0000-000000000000",
  "club_type": "football",
  "club_internal_id": "123456",
  "url": "https://www.fotbal.cz/...",
  "logo_url": "https://is1.fotbal.cz/media/kluby/.../logo.jpg",
  "address": "Milady Horákové 98, 160 00 Praha 6",
  "category": "Muži A",
  "competitions": [
    {
      "id": "12345",
      "code": "1. LIGA",
      "name": "Fortuna Liga",
      "team_count": "16",
      "matches_link": "https://www.fotbal.cz/...",
      "matches": [
        {
          "date_time": "12.08.2023 18:00",
          "home": "AC Sparta Praha",
          "home_id": "00000000-0000-0000-0000-000000000000",
          "home_logo_url": "https://.../sparta.png",
          "away": "SK Slavia Praha",
          "away_id": "11111111-1111-1111-1111-111111111111",
          "away_logo_url": "https://.../slavia.png",
          "score": "2:1",
          "venue": "Stadion Letná",
          "match_id": "match12345",
          "report_url": "https://www.fotbal.cz/..."
        }
      ]
    }
  ]
}

Club Tables (Standings)
GET /club/{type}/{id}/table - id must be provided in the setup of the club

Returns standings (overall table) for each competition of the club.

Example: https://facr.tdvorak.dev/club/football/00000000-0000-0000-0000-000000000000/table

Response shape
{
  "name": "AC Sparta Praha",
  "club_id": "00000000-0000-0000-0000-000000000000",
  "club_type": "football",
  "club_internal_id": "123456",
  "url": "https://www.fotbal.cz/...",
  "logo_url": "https://is1.fotbal.cz/media/kluby/.../logo.jpg",
  "competitions": [
    {
      "id": "12345",
      "code": "1. LIGA",
      "name": "Fortuna Liga",
      "team_count": "16",
      "matches_link": "https://www.fotbal.cz/...",
      "table": {
        "overall": [
          {
            "rank": "1",
            "team": "AC Sparta Praha",
            "team_id": "00000000-0000-0000-0000-000000000000",
            "team_logo_url": "https://.../sparta.png",
            "played": "10",
            "wins": "8",
            "draws": "2",
            "losses": "0",
            "score": "25:5",
            "points": "26"
          },
          {
            "rank": "2",
            "team": "SK Slavia Praha",
            "team_id": "11111111-1111-1111-1111-111111111111",
            "team_logo_url": "https://.../slavia.png",
            "played": "10",
            "wins": "7",
            "draws": "2",
            "losses": "1",
            "score": "20:8",
            "points": "23"
          }
        ]
      }
    }
  ]
}