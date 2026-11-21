# Vendored: age word list

Source: https://github.com/FiloSottile/age
Commit: b74dce4cdbe35b5e5f66c06d9612b72f89028758
License: BSD-3-Clause (`LICENSE` beside this file)

`internal/backups/archive/words.txt` is the word list from age's `cmd/age/wordlist.go` (v1.3.2), copyright 2019 The age Authors, one word per line. It is BIP39's 2048 English words, which age uses to suggest passphrases. SlipStream draws its Backup passphrases from it, because the list sits in age's `package main` and can't be imported.
