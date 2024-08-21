FROM opensuse/leap:15.5

RUN useradd -m tux
RUN groupadd -g 2001 admin && usermod -G admin tux  && echo '%admin ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers

RUN zypper in -y pcre2-devel gcc-c++ clang cmake git ninja tar curl gzip python39 go1.22 bash-completion \
                gcc automake bzip2 libbz2-devel xz xz-devel openssl-devel ncurses-devel \
                readline-devel zlib-devel tk-devel libffi-devel sqlite3-devel gdbm-devel make findutils patch
COPY . /home/tux/dev

RUN git clone https://github.com/sekiguchi-nagisa/arsh.git
RUN mkdir -p ./arsh/build
RUN cd arsh && cd build && cmake .. -G Ninja && cmake --build . && cmake --build . -- install

WORKDIR /home/tux/dev/
CMD ls ./ && go build -buildvcs=false -v ./... && \
    RUN_IN_CONTAINER=on arsh ./test/e2e/pyenv-virtualenv/test_pyenv-virtualenv.arsh
