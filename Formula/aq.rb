class Aq < Formula
  desc "antiQuarantine — CLI to remove com.apple.quarantine xattr from files"
  homepage "https://github.com/jurek-zsl/antiQuarantine/"
  url "https://github.com/jurek-zsl/homebrew-antiQuarantine/releases/download/v1.0.0/aq"
  sha256 "4afa6033db6ab3834d00df4739c27c45582bdf379eb7112e968f461d896d75e1"
  version "1.0.0"

  def install
    bin.install "aq"
  end

  def caveats
    <<~EOS
      Thanks for installing aq!
      Example:
        aq MyApp.app
        aq --remove MyApp.app
        aq --folder ~/Downloads
    EOS
  end
end
