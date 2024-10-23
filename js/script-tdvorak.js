const newMatch = {
    homeTeam: {
      name: "FC Bizoni UH",
      logo: "./img/logo.png",
      score: 0
    },
    awayTeam: {
      name: "Žabinští Vlci Brno B",
      logo: "https://is1.fotbal.cz/media/kluby/59452859-3d56-48ce-826b-121c4ccf8400/59452859-3d56-48ce-826b-121c4ccf8400_crop.jpg",
      score: 0
    },
    date: "Neděle 3. 11. 2024",
    time: "20:00",
    location: "SH Uherské Hradiště",
    matchLink: "zapasy/bizoni-vlci.html"
  };

  // Update content dynamically using JavaScript
  document.getElementById('home-team').textContent = newMatch.homeTeam.name;
  document.getElementById('home-logo').src = newMatch.homeTeam.logo;
  document.getElementById('home-score').textContent = newMatch.homeTeam.score;

  document.getElementById('away-team').textContent = newMatch.awayTeam.name;
  document.getElementById('away-logo').src = newMatch.awayTeam.logo;
  document.getElementById('away-score').textContent = newMatch.awayTeam.score;

  document.getElementById('score').innerHTML = `${newMatch.homeTeam.score} <span>:</span> ${newMatch.awayTeam.score}`;

  document.getElementById('match-date').textContent = `${newMatch.date}, ${newMatch.location}`;
  document.getElementById('match-time').textContent = newMatch.time;
  document.getElementById('match-link').href = newMatch.matchLink;