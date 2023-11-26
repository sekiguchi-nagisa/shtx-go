FROM opensuse/leap:15.5

RUN useradd -m tux
RUN groupadd -g 2001 admin && usermod -G admin tux  && echo '%admin ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers

COPY . /home/tux/dev

RUN zypper in -y pcre2-devel gcc-c++ clang cmake git ninja python3.9 go1.20

RUN git clone https://github.com/sekiguchi-nagisa/ydsh.git
RUN mkdir -p ./ydsh/build
RUN cd ydsh && cd build && cmake .. -G Ninja && cmake --build . && cmake --build . -- install

WORKDIR /home/tux/dev/
CMD ls ./ && ydsh ./script/build_all.ds
