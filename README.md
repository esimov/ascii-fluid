# ascii-fluid

**`ascii-fluid`** is a webcam (face) controlled ASCII fluid simulation running in your terminal. You can control the fluid dynamics with your computer mouse/touchpad but also with your face trough a webcam. 

![ascii-fluid](https://user-images.githubusercontent.com/883386/73605776-2b83bf00-45ab-11ea-93d1-ad6b2a6010e7.gif)

## How it is working?

The fluid solver is mainly based on Jos Stam's paper [Real-Time Fluid Dynamics for Games](https://pdfs.semanticscholar.org/847f/819a4ea14bd789aca8bc88e85e906cfc657c.pdf). This project is using [tcell](https://github.com/gdamore/tcell) library for rendering the fluid simulation in terminal.

## Usage

```bash
 $ go get -u -v github.com/esimov/ascii-fluid
 $ cd wasm & make
```

This will start three new operation simultaneously: it will open a new terminal window, starts a new web server for listening the websocket connection and will build a webassembly interface for accessing the webcam. 

The [Pigo](https://github.com/esimov/pigo) face detection library is used to transfer the first detected face coordinates trough the socket to the terminal app. On each refresh rate (defined as a parameter) the terminal will update the fluid particles.

## Libraries used

- https://github.com/gdamore/tcell
- https://github.com/esimov/pigo
- https://github.com/gorilla/websocket

## Author

* Endre Simo ([@simo_endre](https://twitter.com/simo_endre))

## License

Copyright © 2020 Endre Simo

This software is distributed under the MIT license. See the LICENSE file for the full license text.
