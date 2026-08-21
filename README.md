# CLONE

A quick shortcut for cloning Git repositories.

## Install

Download the latest `.deb` file, then run:

```bash
sudo apt install ./clone_0.1.1_amd64.deb
```

Or build from source:

```bash
git clone https://github.com/StrangeQuark/git-clone-shortcut.git
cd git-clone-shortcut
make build
sudo make install
```

## Set up

When installed interactively with `sudo apt install`, CLONE asks for your
default Git host and username. If you skip that prompt, run:

You can also set them yourself:

```bash
clone init --domain github.com --username StrangeQuark
```

## Use

```bash
clone authservice
# https://github.com/StrangeQuark/authservice

clone testUser/testService
# https://github.com/testUser/testService

clone testWeb.com/testUser/testService
# https://testWeb.com/testUser/testService
```

Useful options:

```bash
clone authservice --ssh
clone authservice --into ~/code/authservice
clone authservice -- --depth 1
```

Change your defaults anytime:

```bash
clone update domain gitlab.com
clone update username another-user
```
