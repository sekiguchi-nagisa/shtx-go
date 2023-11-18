FROM opensuse/leap:15.5

RUN useradd -m tux
RUN groupadd -g 2001 admin && usermod -G admin tux  && echo '%admin ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers

COPY . /home/tux/dev

RUN zypper in -y pcre2-devel gcc-c++ clang cmake git ninja tar curl bzip2 gzip python39 go1.20 bash-completion

RUN git clone https://github.com/sekiguchi-nagisa/ydsh.git
RUN mkdir -p ./ydsh/build
RUN cd ydsh && cd build && cmake .. -G Ninja && cmake --build . && cmake --build . -- install

WORKDIR /home/tux/dev/
CMD ls ./ && go build -buildvcs=false -v ./... && \
    RUN_IN_CONTAINER=on ydsh ./test/e2e/pyenv/test_pyenv.ds
