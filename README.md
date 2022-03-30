<h1 align="center">🌐 Bahnboom 🧰</h1>
<h4 align="center">A CLI that reports current service disurptions and planned
  maintenance for <a href="https://bahnhof.se">Bahnhof</a>.</h4>
<p align="center">
    <a href="https://github.com/daenney/bahnboom/actions/workflows/ci.yaml"><img src="https://github.com/daenney/bahnboom/actions/workflows/ci.yaml/badge.svg" alt="Build Status"></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/daenney/bahnboom" alt="License: MIT"></a>
</p>

Please note that since Bahnhof does not provide an API or RSS feed with
status information this CLI does some trickery to get it from the website.
Should Bahnhof change how they publish and retrieve the data, this CLI will
likely break.

## Build

Builds like any other Go project, with Go >= 1.16:

```
$ go build -o bahnboom
```

## Test

```
$ go test -v
```

## Usage

```
$ ./bahnboom
• 🔥 2022-03-29: Ongoing service disruption on Zitius in Eslöv
• 🔥 2022-03-30: Ongoing service disruption on Lunet in Måttsund
• 👷 2022-03-30: Scheduled maintenance on Kramfors Stadsnät in Nyland
```

See `-help` for other flags.
