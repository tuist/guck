# Changelog

All notable changes to this project will be documented in this file.

## [0.1.2] - 2025-11-22

### 🐛 Bug Fixes

- Hide Git diff metadata lines from UI ([b8d9ab6](b8d9ab6946925116dc325c3e093c8c975a508e8b))

## [0.1.1] - 2025-11-21

### 📚 Documentation

- Update README.md for Go rewrite ([6d490e9](6d490e992ff2f8c2cb5612c9a70256554e81e78d))

## [0.1.0] - 2025-11-21

### 🐛 Bug Fixes

- Resolve compilation errors ([06c517b](06c517ba883ed4656928174cda34b40abe80cf19))
- Remove daemonize dependency and use process spawning instead ([535239f](535239f4f763382a0dcea2692042d93d90672475))
- Set current_dir and add wait time for daemon spawning ([1cdc1cf](1cdc1cf63d3335dc7346e0a5298dac9f129e2afa))
- Ensure Config::default() uses default_base_branch ([1fe1dc8](1fe1dc89646b5f9513831846c2691de6a4e10328))
- Resolve borrow checker error with base_branch_override ([53535d3](53535d3e770008f778b87580c3cade043229b65e))
- Use nohup for daemon spawning on Unix ([5bbd5c4](5bbd5c40b4c1433c3cb8b79e3eeb23b057956ee4))
- Check actual port availability before allocating ([738f1c0](738f1c00f89c14354955b3fe8b8249c1e23d0ba7))

### 🚀 Features

- Initial implementation of guck CLI ([781f0ae](781f0aef1c82e35ce014d46b05377796d748e82f))
- [**breaking**] Implement daemon-based architecture with auto-start ([c63f411](c63f41192b6cd8a103a633c08f1969e7322cb8ce))
- Use random port allocation for daemon processes ([edd1b34](edd1b34e0be7dfa1b494426e853838579f80e64b))
- Implement random port allocation in daemon manager ([2e1bbfc](2e1bbfcc74d5924a640569b43850f0f2399350b9))
- Add 'guck start' command to run server in foreground ([bff5e31](bff5e314791cd1afd973d5dd2f25df0ae72f3602))
- Reimplement guck in Go ([3d4e5bb](3d4e5bbc30908091646e2ae6bf8b583e83323c09))


