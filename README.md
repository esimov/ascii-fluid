# ascii-fluid

**ascii-fluid** is a webcam (face) controlled ASCII fluid simulation running in your terminal. You can control the fluid dynamics with your computer mouse/touchpad but also with your face trough a webcam.

![ascii-fluid](https://user-images.githubusercontent.com/883386/73605776-2b83bf00-45ab-11ea-93d1-ad6b2a6010e7.gif)


## Usage
```bash
 $ go get -u -v github.com/esimov/ascii-fluid
 $ cd wasm && make
```

## How it is working?

The fluid solver is mainly based on Jos Stam's paper [Real-Time Fluid Dynamics for Games](https://pdfs.semanticscholar.org/847f/819a4ea14bd789aca8bc88e85e906cfc657c.pdf). The [tcell](https://github.com/gdamore/tcell) library is used for rendering the fluid simulation in terminal and [gorrilla/websocket](https://github.com/gorilla/websocket) package for communicating trough a websocket connection with the [Pigo](https://github.com/esimov/pigo) face detection library running in Webassembly.

This will start three new operation simultaneously:
- it will open a new terminal window
- it will start a new web server for listening the websocket connection and
- will build a webassembly interface for accessing the webcam.

The coordinates of the first detected face will be transferred over the websocket connection to the terminal application. On each refresh rate (defined as a parameter) the terminal will update the fluid particles.

## Libraries used

- https://github.com/gdamore/tcell
- https://github.com/esimov/pigo
- https://github.com/gorilla/websocket

## Author

* Endre Simo ([@simo_endre](https://twitter.com/simo_endre))

## License

Copyright © 2020 Endre Simo

This software is distributed under the MIT license. See the LICENSE file for the full license text.
