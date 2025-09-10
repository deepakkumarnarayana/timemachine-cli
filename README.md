# TimeMachine CLI ⏰

**Your AI coding safety net - Never lose working code to AI experiments again!**

> *"I asked ChatGPT to refactor my authentication system... and it completely broke everything. I wish I could just go back 10 minutes."*

Sound familiar? **TimeMachine CLI** solves this exact problem! 

It automatically creates snapshots of your code **every time you make changes**, so when AI breaks something (and let's be honest, it happens), you can instantly rollback to when everything was working perfectly.

**The best part?** It's completely invisible to your normal Git workflow. No messy commits, no cluttered history, just pure peace of mind.

[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-linux%20tested%20%7C%20mac%20%26%20windows%20available-orange.svg)](#works-on-your-platform)

*Currently battle-tested on Linux, with Mac and Windows builds ready to go!*

## 🎯 The Real Problem We're Solving

**You know this feeling...**

You're in the zone, your code is working perfectly, then you ask Claude or ChatGPT to "just make a small improvement" and suddenly:

- ❌ Your app won't start
- ❌ Tests are failing everywhere  
- ❌ You can't remember exactly what changed
- ❌ `git diff` shows 47 files modified
- ❌ You're frantically trying to undo things manually

**What if you could just say "go back to 5 minutes ago" and everything works again?**

That's exactly what TimeMachine does! Think of it like automatic save points in a video game, but for your code.

### Why TimeMachine is Perfect for AI Development

- 🛡️ **Fearless Experimentation** - Try anything, rollback instantly if it breaks
- ⚡ **Lightning Fast** - No waiting, no complex commands, just instant recovery
- 🔒 **Git-Safe** - Never touches your Git history, staging area, or commits
- 🎯 **AI-Aware** - Captures context across branches and AI interactions
- 🧠 **Smart Grouping** - Bundles related changes together automatically

## 🚀 See It In Action (2-Minute Setup)

**Ready to try it? Here's exactly what you'll experience:**

### Step 1: One Command to Start
```bash
# In your project directory
timemachine init
```
```
✅ TimeMachine initialized! 
✅ Shadow repository created at .git/timemachine_snapshots/
✅ Added to .gitignore (won't affect your main repo)
✅ Ready to protect your code!
```

### Step 2: Start the Magic
```bash
timemachine start
```
```
🔍 Watching for changes in your project...
📁 Monitoring: src/, tests/, package.json, and 247 other files
⏰ Ready! All your changes will now be automatically captured.

Press Ctrl+C to stop watching.
```

*That's it! TimeMachine is now silently protecting you in the background.*

### Step 3: Work Fearlessly 
```bash
# Now just code normally...
# Ask AI to refactor something...
# Make experimental changes...
# Everything is automatically saved!
```

### Step 4: When Something Breaks (The Magic Moment)
```bash
timemachine list
```
```
📸 Your recent snapshots:

2 minutes ago    abc1234  "AI refactored auth system" (12 files changed)
8 minutes ago    def5678  "Added user validation" (3 files changed)  
15 minutes ago   ghi9012  "Working login feature" (8 files changed) ← Your working code!
```

```bash
# Go back to when everything worked!
timemachine restore ghi9012
```
```
⚠️  This will restore ALL files from this snapshot
   Any uncommitted changes will be lost!

✅ Restored 8 files to snapshot ghi9012
✅ Your code is back to working state!
✅ (Git history untouched - you can still commit when ready)
```

**What `restore` does:** Changes your working files back to the snapshot state (like "undo" for your whole project). Your Git commits and staging area stay exactly the same.

**Boom! Crisis averted. Time to try again.** 🎉

## 📦 Get TimeMachine (Super Easy!)

**Choose your preferred method - they're all simple:**

### 🎯 Option 1: One-Line Install (Easiest)

**Linux & Mac users:**
```bash
curl -fsSL https://raw.githubusercontent.com/deepakkumarnarayana/timemachine-cli/main/install.sh | bash
```

This magical script will:
- Figure out your system automatically
- Download the right version for you  
- Put it in the right place
- Make sure it's ready to use

*Takes about 10 seconds and you're done!*

### 🎯 Option 2: Download and Go

Grab your version from [GitHub Releases](https://github.com/deepakkumarnarayana/timemachine-cli/releases):

- **Linux**: `timemachine-linux-amd64` (most common)
- **Mac Intel**: `timemachine-macos-amd64`
- **Mac Apple Silicon**: `timemachine-macos-arm64`
- **Windows**: `timemachine-windows-amd64.exe`

**Linux/Mac Example:**
```bash
# Download it
wget https://github.com/deepakkumarnarayana/timemachine-cli/releases/latest/download/timemachine-linux-amd64

# Make it runnable
chmod +x timemachine-linux-amd64

# Put it somewhere useful (optional - needs sudo)
sudo mv timemachine-linux-amd64 /usr/local/bin/timemachine

# Or just run it from current folder
./timemachine-linux-amd64 init
```

### 🎯 Option 3: Go Install (For Go Developers)

**If you have Go installed (easiest for developers):**
```bash
go install github.com/deepakkumarnarayana/timemachine-cli/cmd/timemachine@latest
```

This method:
- ✅ **Works on all platforms** (Linux, Mac, Windows)
- ✅ **Always gets you the latest version** 
- ✅ **Automatically puts binary in your PATH**
- ✅ **No manual downloading or moving files needed**

*Perfect for Go developers who want the simplest install!*

### 🎯 Option 4: Build It Yourself

*For developers who like to build from source:*
```bash
git clone https://github.com/deepakkumarnarayana/timemachine-cli.git
cd timemachine-cli/timemachine
go build -o timemachine ./cmd/timemachine
```

## Works on Your Platform

- **🐧 Linux**: Fully tested and rock-solid 
- **🍎 Mac**: Should work great (binaries ready, just needs testing!)
- **🪟 Windows**: Available but seeking brave testers

*TimeMachine is built with Go, so it should work everywhere Go works. Linux is just where we've done the most testing.*

**Got it working on Mac or Windows? [Let us know!](https://github.com/deepakkumarnarayana/timemachine-cli/issues) We'd love to hear about it.**

## 🧠 "But How Does It Actually Work?" (For the Curious)

**The secret sauce: TimeMachine creates a hidden "shadow" repository that watches your same files but keeps its own separate history.**

Think of it like this:
```
your-awesome-project/
├── .git/                       # Your normal Git repo (unchanged!)
├── .git/timemachine_snapshots/ # Secret backup repo (auto-hidden)
├── src/                        # Your code files (watched by both)
├── package.json               # (both repos see the same files)
└── ...                        # (but keep separate histories)
```

**What this means for you:**
- ✅ Your normal `git` commands work exactly the same
- ✅ Zero impact on performance or workflow  
- ✅ TimeMachine snapshots are completely separate
- ✅ You can `git commit` and restore from snapshots independently
- ✅ Automatic cleanup keeps things tidy

*It's like having a parallel universe Git repo that doesn't interfere with your main one!*

## 🎮 All the Commands (They're Really Simple)

### The Basics (You'll Use These Most)
```bash
timemachine init     # Set up protection for this project
timemachine start    # Start watching (run in background)
timemachine list     # See what snapshots you have
timemachine restore abc1234  # Go back to snapshot abc1234
```

### When You Need More Details
```bash
timemachine show abc1234     # See exactly what changed in a snapshot
timemachine status           # Check if TimeMachine is working
```

### Cleanup (Occasional Maintenance) 
```bash
timemachine clean --older-than 1w  # Delete snapshots older than 1 week
timemachine clean --older-than 3d  # Delete older than 3 days
```

*That's literally it! Most of the time you'll just use `list` and `restore`.*

## 🎯 Real-World Scenarios (When TimeMachine Saves the Day)

### Scenario 1: "The Refactoring Disaster"
```bash
You: "Claude, can you refactor this messy controller file?"
Claude: *confidently breaks 47 files*

📸 Before: "Working user dashboard" (32 files, 2 min ago)
📸 After:  "AI refactored controller" (47 files, just now) ← BROKEN

$ timemachine restore [before]  # Back to working in 2 seconds!
```

### Scenario 2: "The Dependency Nightmare" 
```bash
You: "ChatGPT, update my package.json dependencies"
ChatGPT: *updates everything to latest, breaks compatibility*

📸 Before: "Stable build with React 17" (package.json, 5 min ago)
📸 After:  "Updated all deps to latest" (package.json + lock, just now) ← WON'T BUILD

$ timemachine restore [before]  # Sanity restored!
```

### Scenario 3: "The Mysterious Branch Mixup"
```bash
# Working on feature branch
$ git checkout main  
📸 [feature→main] BRANCH SWITCH: Back to main branch

# Ask AI to add something
📸 [main] AI added analytics (12 files)

# Oh no! This was supposed to go in the feature branch!
$ git checkout feature
$ timemachine restore [analytics-snapshot]  # Move AI changes to right branch
```

*TimeMachine is smart enough to know when you switch branches and captures that context!*

### Scenario 4: "The 'Small Change' That Wasn't"
```bash
You: "Just add some error handling to this one function"
AI: *rewrites error handling across entire codebase*

📸 Recent snapshots:
- 30 sec ago: "AI improved error handling" (23 files) ← Went too far!
- 10 min ago: "Working payment flow" (8 files) ← This was perfect
- 20 min ago: "Added user validation" (3 files)

$ timemachine restore [working-payment-flow]  # Back to the sweet spot
```

## ⚙️ Fine-Tuning (Optional Customization)

*TimeMachine works great out of the box, but here are some tweaks if you want them:*

### Tell TimeMachine to Ignore Stuff

Sometimes you don't want snapshots of everything (like `node_modules` or build files). Just create a `.timemachine-ignore` file:

```bash
# In your project root
touch .timemachine-ignore
```

Then add patterns just like `.gitignore`:
```gitignore
# Don't snapshot these folders
node_modules/
dist/
build/
.next/

# Or these file types  
*.log
*.cache
.DS_Store

# Or specific files
config/secrets.env
```

*Pro tip: TimeMachine already ignores common stuff automatically, so you might not need this at all!*

## 🆘 "Help! Something's Not Working!"

**Don't worry! Here are the most common fixes:**

### "It's Not Watching My Files!"
```bash
# Check if it's actually running
timemachine status

# See what it's doing in detail
timemachine status --verbose

# Check if your files are being ignored
cat .timemachine-ignore
```

*Most likely: You have files ignored or TimeMachine isn't running.*

### "Too Many Snapshots / Running Slow!"
```bash
# Clean up old snapshots (older than 1 week)
timemachine clean --older-than 1w

# Ignore big directories that change a lot
echo "node_modules/" >> .timemachine-ignore
echo "dist/" >> .timemachine-ignore
```

*TimeMachine automatically bundles quick changes, but cleaning up helps!*

### "Linux: File Watching Stopped Working"
```bash
# Your project might be too big for Linux's default limits
echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf
sudo sysctl -p

# Then restart TimeMachine
timemachine start
```

*This is only needed for huge projects (like monorepos).*

### Still Stuck?

**TimeMachine is safe and simple - if something's weird:**

1. **Stop it**: Press Ctrl+C if it's running
2. **Check status**: `timemachine status` 
3. **Try again**: `timemachine start`
4. **Ask for help**: [Open an issue](https://github.com/deepakkumarnarayana/timemachine-cli/issues)

*Remember: TimeMachine only changes your working files when you restore - your Git history stays untouched!*

## 🛡️ "Is This Safe?" (Security & Privacy)

**Yes! Here's what TimeMachine does and doesn't do:**

✅ **What it does:**
- 📁 **Watches files** and saves snapshots locally
- 💾 **Restores working files** when you ask (like `git restore`)
- 🗂️ **Uses separate storage** (`.git/timemachine_snapshots/`)

❌ **What it never does:**
- 🚫 **Never touches Git history** - your commits stay untouched
- 🚫 **Never modifies staging area** - your `git add` stays intact  
- 🚫 **Never sends data anywhere** - everything stays on your machine
- 🚫 **Never logs secrets** - no passwords or sensitive data stored

*Think of restore like "undo" in your editor, but for your entire project.*

## 🤝 Want to Help Make TimeMachine Better?

**We'd love your help! Here's how:**

### Test It On Your Platform
- ✅ **Linux users**: You're all set!  
- 🍎 **Mac users**: Try it and let us know how it works!
- 🪟 **Windows users**: Be a pioneer and test it!

### Report Issues or Ideas
Found a bug? Have an idea? [Tell us about it!](https://github.com/deepakkumarnarayana/timemachine-cli/issues)

### Code Contributions
```bash
git clone https://github.com/deepakkumarnarayana/timemachine-cli.git
cd timemachine-cli/timemachine
make test  # Make sure everything works
# Make your changes and submit a PR!
```

*We're friendly and welcoming to all skill levels!*

---

## 🚀 Ready to Code Fearlessly?

**TimeMachine CLI gives you AI superpowers without the AI risk.**

Never again will you think:
- *"I wish I could just go back 5 minutes..."* 
- *"What exactly did ChatGPT change?"*
- *"I should make a commit before trying this..."*
- *"Maybe I shouldn't ask AI to help with this..."*

**Instead, you'll think:**
- *"Let's try this wild idea - I can always rollback!"* ✨
- *"AI, refactor everything - I'm protected!"* 💪
- *"Let me experiment with this approach..."* 🧪

### Get Started Right Now:
```bash
# Install TimeMachine
curl -fsSL https://raw.githubusercontent.com/deepakkumarnarayana/timemachine-cli/main/install.sh | bash

# Set it up in your project
cd your-project
timemachine init

# Start the magic
timemachine start
```

**That's it! You're now coding with a safety net.** 🎉

*Happy AI-assisted coding! 🤖✨*

---

**TimeMachine CLI** - *MIT License* - [GitHub](https://github.com/deepakkumarnarayana/timemachine-cli) - Made with ❤️ for fearless developers