# Track Scroll

**tags:** mouse, trackball, scroll, wheel

## Summary

This app is based on the idea of https://github.com/martin-stone/touchcursor.
We install a global mouse hook and translate mouse movement into scroll (wheel) commands.
This doesn't necessarily make sense for most users, but is very helpful when using a trackball.
The trackball then effectively becomes a scroll wheel.
The current trigger to switch modes is to hold down both mouse buttons.

## Implementation

Only targets Windows 10 x64.

The implementation mostly revolves around calling the correct Windows API functions.
We use https://github.com/moutend/go-hook to facilitate installing the global mouse hook.
Triggering the simulated mouse events is based on the code in https://github.com/micmonay/keybd_event.
