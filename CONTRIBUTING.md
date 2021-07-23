# Contributing

These notes are mostly intended for myself :).

## Release process

```
vim version.txt
git add version.txt
git commit -m "Release $(cat version.txt)"
git tag $(cat version.txt)
git push
git push --tags
```
