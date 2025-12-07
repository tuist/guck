# Changelog

All notable changes to this project will be documented in this file.

## [1.7.1] - 2025-12-07

### 📦 Dependency Updates

- **(deps)** Bump golang.org/x/crypto from 0.21.0 to 0.45.0 (#24) ([9d7a0a1](9d7a0a152c747b852752b94f667f459a373788c4))

## [1.7.0] - 2025-12-07

### 🚀 Features

- Improve diff styling to match GitHub's design (#25) ([3041d0e](3041d0e7c1df02859139e2ac186367beab982f5b))

## [1.6.0] - 2025-12-06

### 📦 Dependency Updates

- **(deps)** Bump github.com/go-git/go-git/v5 from 5.12.0 to 5.13.0 (#23) ([5f28d31](5f28d31a8e572c3a47f2d90897c94896e0809734))

### 🚀 Features

- Display uncommitted changes (staged and unstaged) (#22) ([5cc5d3e](5cc5d3e6f4cb02d3d9bdeb17cce483090368f3cd))

## [1.5.1] - 2025-11-24

### 🐛 Bug Fixes

- Compare against remote tracking branch and merge-base (#19) ([e143e41](e143e41aaff7b4b979a3a590af25c3320c97ac5e))

## [1.5.0] - 2025-11-24

### 🚀 Features

- Apply syntax highlighting to added and deleted lines (#18) ([b232283](b232283714474c93108e66339000a7f895d70091))

## [1.4.0] - 2025-11-24

### 🚀 Features

- Add CLI commands for comments and notes with multiple output formats (#17) ([ef00367](ef0036749cf5c98ccbcf42eb66b15c721abeb494))

## [1.3.2] - 2025-11-24

### 🐛 Bug Fixes

- Resolve syntax highlighting theme conflict causing dark comments (#16) ([4820835](4820835ad68e5adc149767831e1d09df4ae6e4c8))

## [1.3.1] - 2025-11-24

### 🐛 Bug Fixes

- Exclude deleted lines from line number count in diff view (#15) ([d700bc2](d700bc2321545c29e2aa6ad2836d263015d534f9))

## [1.3.0] - 2025-11-24

### 🚀 Features

- Improve shell integration message (#14) ([832d4c8](832d4c85f73942ed4c696fc7a9f207f9d61541a6))

## [1.2.0] - 2025-11-24

### 🚀 Features

- Add UI visualization for AI agent notes (#13) ([8c63e98](8c63e98e0cd5039402891cd15bb0069c83cc9d34))

## [1.1.0] - 2025-11-24

### 🚀 Features

- Add AI agent notes for code explanations ([522f9bb](522f9bb4c14157b26432d599b6b540654d2d2f65))

## [1.0.0] - 2025-11-23

### 🚀 Features

- [**breaking**] Make repo_path required in MCP tools and simplify MCP command ([073aa9d](073aa9d434c808ead84e6576d13a82acfdea21cb))

## [0.6.0] - 2025-11-23

### 🚀 Features

- Preserve diff prefixes (+/-) in syntax-highlighted code ([ddbd584](ddbd584ae7d952da21ac759f33ba2338e36d4b89))

## [0.5.0] - 2025-11-23

### 🚀 Features

- Display repository name in page title from remote origin ([18569f8](18569f86337f423661460473619b5f1069f92903))

## [0.4.0] - 2025-11-23

### 🚀 Features

- Auto-stop daemon when leaving git repository ([6fa3447](6fa3447505c89d901cdc0f828380232de98419f4))

## [0.3.4] - 2025-11-23

### 🐛 Bug Fixes

- Make comment text white in light mode ([983dda8](983dda8b1c7e6fc7814c1c656b7da51ef3ccddd2))

### 📚 Documentation

- Update README.md [skip ci] ([8f24598](8f24598c1a3da6fde77aad7ca800bccf68da0c95))
- Create .all-contributorsrc [skip ci] ([2a12d52](2a12d521cc06dde5de7dd990da479d6b8bf4924b))

## [0.3.3] - 2025-11-22

### 🐛 Bug Fixes

- Change Guck title to link to GitHub repository ([b87d3bc](b87d3bc5fa446a3b42c34eb727b996b99711de6b))

## [0.3.2] - 2025-11-22

### 🐛 Bug Fixes

- Use runtime.GOOS for browser detection instead of version checks ([90d6dff](90d6dffb3ffbf4d7252fb5ab211d33da531731ef))
- Add missing runtime import ([95ce3cb](95ce3cb895da87011e3f1c9c95c12164bbe7afd5))

## [0.3.1] - 2025-11-22

### 📚 Documentation

- Recommend using mise for MCP server configuration ([11cfd94](11cfd9429cff21f08675c999eb495782f63c8fe3))

## [0.3.0] - 2025-11-22

### 🚀 Features

- Add MCP server support with comprehensive tests and improved CI ([a31bc59](a31bc59f3427c5424ac9d121826aacb8582c8fe6))

## [0.2.0] - 2025-11-22

### 🚀 Features

- Add colorful CLI output ([8a06b2a](8a06b2a7fbe5125509997d0d4500c30315277118))

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


