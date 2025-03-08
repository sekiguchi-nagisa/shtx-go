FROM opensuse/tumbleweed

RUN zypper in -y pcre2-devel git go1.22 diffutils gcc-c++ clang cmake ninja re2c

COPY . /home/tux/dev

RUN git clone https://github.com/sekiguchi-nagisa/arsh.git
RUN mkdir -p ./arsh/build
RUN cd arsh && cd build && cmake .. -G Ninja && cmake --build . && cmake --build . -- install

WORKDIR /home/tux/dev/
CMD ls ./ && ./scripts/build.sh && \
    arsh ./test/run_all_tests.ds /usr/local/share/arsh/tools/litecheck
