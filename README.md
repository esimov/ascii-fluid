# 🌊 ascii-fluid

**ascii-fluid** is a face controlled ASCII fluid simulation running real time in your terminal. You can control the fluid dynamics with your face by using a webcam, but also with the mouse or touchpad.

![ascii-fluid](https://user-images.githubusercontent.com/883386/73605776-2b83bf00-45ab-11ea-93d1-ad6b2a6010e7.gif)


## Usage
```bash
 $ go get -u -v github.com/esimov/ascii-fluid
 $ cd wasm && make
```

## How it is working?

The fluid solver is mainly based on Jos Stam's paper [Real-Time Fluid Dynamics for Games](https://pdfs.semanticscholar.org/847f/819a4ea14bd789aca8bc88e85e906cfc657c.pdf). The [tcell](https://github.com/gdamore/tcell) library is used for rendering the fluid simulation in terminal and [gorrilla/websocket](https://github.com/gorilla/websocket) package for communicating through a websocket connection with the Webassembly version of the [Pigo](https://github.com/esimov/pigo) face detection library.

This will start three new operation simultaneously:
- it will open a new terminal window
- it will start a new web server for listening to the websocket connection
- it will build the webassembly interface for accessing the webcam.

The coordinates of the first detected face will be transferred over the websocket connection to the terminal application. On each refresh rate (defined as a parameter) the terminal will update the fluid simulation.

## OS Support
**This program has been tested on Linux and MacOS, but normally it should also run on Windows.**

Because of the OS imposed security constrains there are some important steps you need to take:

#### MacOS:
In MacOS you must set the accessibility authorization for the terminal you are running from.

<img src="https://user-images.githubusercontent.com/705503/80077645-11c09b00-854e-11ea-8b52-ad130b42028b.png" width=300/>

## Libraries used

- https://github.com/gdamore/tcell
- https://github.com/esimov/pigo
- https://github.com/gorilla/websocket

## Author

* Endre Simo ([@simo_endre](https://twitter.com/simo_endre))

## License

Copyright © 2020 Endre Simo

This software is distributed under the MIT license. See the LICENSE file for the full license text.
