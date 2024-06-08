FROM opensuse/leap:15.5

RUN useradd -m tux
RUN groupadd -g 2001 admin && usermod -G admin tux  && echo '%admin ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers

RUN zypper in -y pcre2-devel gcc-c++ clang cmake git ninja direnv python39 go1.21

COPY . /home/tux/dev

RUN git clone https://github.com/sekiguchi-nagisa/arsh.git
RUN mkdir -p ./arsh/build
RUN cd arsh && cd build && cmake .. -G Ninja && cmake --build . && cmake --build . -- install

WORKDIR /home/tux/dev/
CMD ls ./ && go build -buildvcs=false -v ./... && \
    RUN_IN_CONTAINER=on arsh ./test/e2e/direnv/test_direnv.arsh
