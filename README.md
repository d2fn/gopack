# GoPack

Simple go package management a la [rebar](https://github.com/basho/rebar).

A configuration file tells goop about your dependencies and which version should be included. You can point to a tag, a branch, or, if you are being naughty, master. The programming community would thank you not to carry out such a travesty as it leaves your code open to breaking changes. Much better to point at _immutable_ code.

```
[dependencies]
	[github.com/user/project1]
		tag = 1.0
	[github.com/user/project2]
		branch = foo
```

Then simply run, install, and test your code much as you would have with the ```go``` command. Just replace ```go``` with ```gp```.

```gp test```
```gp run *.go```

etc…

The ```gp``` command will make sure your dependencies are downloaded, their respective git repos are pointed at the appropriate tag or branch, and your code is compiled against the desired library versions. Project dependencies are stored locally in the ```vendor``` directory.

