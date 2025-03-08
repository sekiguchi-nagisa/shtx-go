FROM opensuse/tumbleweed

RUN zypper in -y git go1.23 diffutils

COPY . /home/tux/dev
RUN zypper addrepo --no-gpgcheck -f https://download.opensuse.org/repositories/home:nsekiguchi/openSUSE_Tumbleweed/home:nsekiguchi.repo && \
    zypper refresh && \
    zypper install -y arsh

WORKDIR /home/tux/dev/
CMD ls ./ && go build -buildvcs=false -v ./... && \
    arsh ./test/run_all_tests.ds /usr/share/arsh/tools/litecheck
