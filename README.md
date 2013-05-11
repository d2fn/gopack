# gopack

Simple go package management a la [rebar](https://github.com/basho/rebar).

A configuration file tells gopack about your dependencies and which version should be included. You can point to a tag, a branch, or, if you are being naughty, master. The programming community would thank you not to carry out such a travesty as it leaves your code open to breaking changes. Much better to point at _immutable_ code.

```toml
[deps.memcache]
import = "github.com/bradfitz/gomemcache/memcache"
tag = "1.2"

[deps.mux]
import = "github.com/gorilla/mux"
branch = "1.0rc2"

[deps.toml]
import = "github.com/pelletier/go-toml"
commit = "23d36c08ab90f4957ae8e7d781907c368f5454dd"
```

Then simply run, install, and test your code much as you would have with the ```go``` command. Just replace ```go``` with ```gp```.

```gp test```
```gp run *.go```

etc…

The ```gp``` command will make sure your dependencies are downloaded, their respective git repos are pointed at the appropriate tag or branch, and your code is compiled against the desired library versions. Project dependencies are stored locally in the ```vendor``` directory.

# Installation

First checkout and build from source
```
git clone git@github.com:d2fn/gopack.git
cd gopack
go get
go build
```

Then copy the ```gopack``` binary to your project directory and invoke just as you would go. Make sure the current directory is on your path or place the ```gp``` binary elsewhere on your path.
```
cp gopack ~/projects/mygoproject/gp
cd ~/projects/myproject
gp run *.go
```

