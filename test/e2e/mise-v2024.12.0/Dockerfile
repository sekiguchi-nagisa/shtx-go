FROM opensuse/leap:15.5

RUN useradd -m tux
RUN groupadd -g 2001 admin && usermod -G admin tux  && echo '%admin ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers

RUN zypper in -y pcre2-devel gcc-c++ clang cmake git ninja python39 go1.22 tar curl gzip bash-completion

COPY . /home/tux/dev

RUN git clone https://github.com/sekiguchi-nagisa/arsh.git
RUN mkdir -p ./arsh/build
RUN cd arsh && cd build && cmake .. -G Ninja && cmake --build . && cmake --build . -- install

WORKDIR /home/tux/dev/
CMD ls ./ && git config --global --add safe.directory "${PWD}" && ./scripts/build.sh && \
    RUN_IN_CONTAINER=on MISE_VERSION=v2024.12.0 arsh ./test/e2e/mise-v2024.12.0/test_mise.arsh
