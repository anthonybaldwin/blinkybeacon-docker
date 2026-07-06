# Blinkybeacon

_A set of utilities for working with beacon lights, currently just the USB one included with the Farming Simulator 22 Collector's Edition._

<img src="blinkybeacon-docker.jpg" alt="blinkybeacon-docker" width="75%">

## Why on earth

Were you lucky enough to have the Farming Simulator 22 Collector's Edition magically appear on your desk?
If so, you'll already appreciate the fact that it has a super cool USB beacon included - 
and the best part about it is that it synchronises with the in-game tractor! The immersion factor is truly off the scale.

But this begs the question: if the game can make the super cool siren blink and spin, why can't _we_ hook that into whatever else takes our fancy?

Here's some really silly ideas:
* Have the beacon go off whenever your local sportsball team scores a goal
* Make the beacon spin when your server is down or on fire
* Strobe the beacon when it's time for a dance party

The list of possibilities is, quite simply and much like farming itself, endless.

## This repository

This repo contains a package called `fsbeacon` for interfacing directly with connected USB beacons,
as well as a command package also called `fsbeacon` for performing simple commands from the command line (I'm not sure this sentence has the word _command_ in it enough. Command.)

Right now, all of this is dependent on [_hidapi_](https://github.com/libusb/hidapi) and its limitations (namely - Windows, Linux, macOS, and possibly other systems via libusb).

To use the command, first make the binary, or download it from releases:

`make`

Then you can do things like the following:

```
fsbeacon strobe 15
fsbeacon spin 2.5
```

If you want the beacon to do a thing indefinitely, simply pass no duration argument - it'll then strobe or spin until you stop the process somehow (probably via CTRL+C).

You'll find more details on using the package in your own applications in its [readme](pkg/fsbeacon/README.md). Please do let me know what you end up making!

## Docker & HTTP API

This fork adds a small HTTP API (`cmd/beacon-api`) and a Docker image that runs it, so you can poke the beacon over the network:

```
curl http://localhost:9100/strobe/15
curl http://localhost:9100/spin/2.5
curl http://localhost:9100/off
```

Durations accept plain seconds (`2`, `2.5`) or Go duration strings (`2s`, `500ms`, `1m`) and must be greater than zero. Omit the duration (e.g. `/spin`) to run until you call `/off`. There's no hard minimum, but expect anything under about half a second to be dominated by USB/process overhead and the beacon's own response time.

Build and run with compose (the beacon device is passed through via `devices`):

```
docker compose up -d
```

Images are published to GHCR by the `docker-build` GitHub Action on pushes to `main`.

## License

This software is licensed under the [MIT license](LICENSE).