FROM opensuse/tumbleweed

RUN zypper in -y pcre2-devel gcc-c++ cmake git ninja go1.21 diffutils

COPY . /home/tux/dev
RUN git clone https://github.com/sekiguchi-nagisa/arsh.git
RUN zypper addrepo https://download.opensuse.org/repositories/home:nsekiguchi/openSUSE_Tumbleweed/home:nsekiguchi.repo && \
    zypper refresh && \
    zypper install arsh

WORKDIR /home/tux/dev/
CMD ls ./ && go build -buildvcs=false -v ./... && \
    arsh ./test/run_all_tests.ds /arsh/tools/litecheck/litecheck
