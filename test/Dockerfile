FROM opensuse/tumbleweed

RUN zypper in -y pcre2-devel gcc-c++ cmake git ninja go1.20 diffutils

COPY . /home/tux/dev
RUN git clone https://github.com/sekiguchi-nagisa/ydsh.git
RUN mkdir -p ./ydsh/build
RUN cd ydsh && cd build && cmake .. -G Ninja && cmake --build . && cmake --build . -- install

WORKDIR /home/tux/dev/
CMD ls ./ && go build -buildvcs=false -v ./... && \
    ydsh ./test/run_all_tests.ds /ydsh/tools/litecheck/litecheck
