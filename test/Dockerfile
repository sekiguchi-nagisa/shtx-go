FROM opensuse/leap:15.5

RUN zypper in -y pcre2-devel gcc-c++ clang cmake git ninja sudo systemd-sysvinit glibc-locale go1.20
RUN useradd -m tux
RUN groupadd -g 2001 admin && usermod -G admin tux  && echo '%admin ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers

COPY . /home/tux/dev
RUN chown -R tux:users /home/tux/dev && chmod -R 744 /home/tux/dev
RUN git clone https://github.com/sekiguchi-nagisa/ydsh.git
RUN mkdir -p ./ydsh/build
RUN cd ydsh && cd build && cmake .. -G Ninja && cmake --build . && cmake --build . -- install

USER tux
WORKDIR /home/tux/dev/
CMD ls ./ && mkdir -p bin && \
    ydsh ./test/run_all_tests.ds
