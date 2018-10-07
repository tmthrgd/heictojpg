# heictojpg

heictojpg small utility to recursively convert HEIC image files to JPGs.

It takes two flags:

- `-recurse` (default `true`) that controls whether or not it descends recursively into sub-directories, and
- `-out` (default `"{dir}{file}.jpg"`) which sets the output path (see newPath in [main.go](/main.go) for the valid tags).

It also accepts a directory to operate in, otherwise it uses the working directory.

It requires [tifig](https://github.com/monostream/tifig) to convert the HEIC file to JPG.