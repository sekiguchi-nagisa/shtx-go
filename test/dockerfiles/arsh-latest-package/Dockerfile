FROM opensuse/tumbleweed

RUN zypper in -y git go1.22 diffutils

COPY . /home/tux/dev
RUN zypper addrepo --no-gpgcheck -f https://download.opensuse.org/repositories/home:nsekiguchi/openSUSE_Tumbleweed/home:nsekiguchi.repo && \
    zypper refresh && \
    zypper install -y arsh

WORKDIR /home/tux/dev/
CMD ls ./ && ./script/build.sh && \
    arsh ./test/run_all_tests.ds /usr/share/arsh/tools/litecheck
