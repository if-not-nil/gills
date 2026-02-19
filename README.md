# gills
lung's userspace utils

all tailored, not painful to compile, easy to modify

## not in this repo but are part of this 
all written in go

- [koghi - a sysfetch tool](https://github.com/if-not-nil/koghi)
- [joshfile - a task runner](https://github.com/if-not-nil/joshfile)
- [horse - a better cd-ls](https://github.com/if-not-nil/horse)

## cat
an advanced cat command. small, has syntax highlighting

supports all flags from cat and some from bat

## choice
pick a random word from file or arguments or get a random number

```bash
[~/gill] go run .
no choices given
usage: choice [flags] [choices]
output: random item(s) from args or stdin (one item per line)

flags:
  -i[N]        random integer from 0 to N (default 1)
  -f[N]        random float from 0 to N (default 1)
  -n K         pick K items (default 1)
  -nu K        pick K unique items (no repeats)
  -s           shuffle: print all choices in random order
  -d DELIM     join output with DELIM instead of newlines
  -x VAL       exclude VAL from choices (repeatable)
  -S SEED      seed for reproducible output
  -c           print count of choices and exit

weighted choices: suffix with :N e.g. 'a:3 b:1'
[~/gill] cat ./dict.txt | choice
ephemeral
[~/gill] cat ./dict.txt | choice
ephemeral
[~/gill] go run . -f10
5.172420807954689
[~/gill] go run . -i10
3
```

## slice
cut a file. your shell gives you functionality to do so from stdio as well

```bash
[~/gill] slice main.go[1:] # or slice main.go [1]
    - whole file starting from the first character 
[~/gill] slice -[1:]
    - standard input from the first character 
[~/gill] slice main.go[1:] -z # zero-indexed
    - whole file starting from the second character
[~/gill] slice main.go[1l:20l]
    - lines 1-20 
[~/gill] slice main.go[1w:20w]
    - words 1-20 
[~/gill] slice main.go[1:20]
    - characters 1-20 
[~/gill] slice main.go[1_3:2_3]
    - characters 1*(10^3)-1*(10^3), so 1000-2000
[~/gill] slice main.go[1b:20b]
    - bytes 1-20 
[~/gill] slice main.go[1kb:20kb]
    - kilobytes 1-20. accepted are kB 
    - accepted are b 512, kB 1000, K 1024, MB 1000*1000, M 1024*1024
```

## new
create/update files/directories with permissions intuitively

```c
// sample -
//
//	new -rwx install - creates a new file called install with rwx perms for all groups
//	new -rw install - creates install with rw-rw-rw-
//	new -rwxRWxrwx install - rwx--xrwx
//	new -Rrxxx install - -wxrwxrwx (R disables read for user, lowercase enable for group+other)
//	new -rwx build/ - creates a build/ dir with rwx for all groups
//	new build/asdf - creates build/ dir and asdf inside with no perms specified (0000)
//	new -rwx src/main/asdf - recursively creates src/ and main/ then asdf with rwxrwxrwx
//	new -x asdf - if asdf exists, adds x for all groups; if new, creates with --x--x--x
//	new -X asdf - removes x from all groups on existing file
//	new -rwx asdf nasdf - creates both asdf and nasdf with rwxrwxrwx
//	new -rwx asdf -rw nasdf - asdf gets rwxrwxrwx, nasdf gets rw-rw-rw-package main
//  new go run . aaa/addsaf/asdf/asd/f - created aaa/addsaf/asdf/asd/f (0000)
```

# not done
### backup
make a backup of a file/directory, optionally compress, use btrfs magic or put in a directory to version with git LFS

### get
copy/move anything from anywhere: your filesystem, whatever curl and rsync support

### bearer
local and self-hosted bare git repo manager, so that you dont have to make something online all of the time

### mail
programs will be able to deposit a letter into a text file

for example, your calendar can say: tell a user they have a meeting scheduled 10h in advance of $DATETIME

which will look like

**~/.mail**
```yaml
a3b:
  from: cal
  message: dont forger that one thing
  action: cal goto event 163
  deadline: Apr 20 1969, 19:00 UTC
  aliveline: Apr 20 1969, 7:00 UTC
  seen: false
```

**commandline**
```
[~] mail get
* until Apr 20 1969, 19:00 UTC
cal:
  dont forger that one thing

  run "mail act a3b" to run command or "mail info a3b" for more

[~] mail get
*empty*

[~] mail get from last
# the message from before

[~] mail get from yesterday
# the message from yesterday

[~] mail get from 01/01 to 31/01
# all from january

[~] cat ./message.yaml | mail send
# and then it gets sent kinda
```
