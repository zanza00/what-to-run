# what-to-run apt repository

APT repository for [what-to-run](https://github.com/zanza00/what-to-run).

## Install

```sh
curl -fsSL https://zanza00.github.io/what-to-run/key.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/what-to-run.gpg
echo 'deb [signed-by=/usr/share/keyrings/what-to-run.gpg] https://zanza00.github.io/what-to-run stable main' \
  | sudo tee /etc/apt/sources.list.d/what-to-run.list
sudo apt update
sudo apt install what-to-run
```
