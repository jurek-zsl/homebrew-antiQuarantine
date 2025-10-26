class Aq < Formula
  desc "antiQuarantine — CLI to remove com.apple.quarantine xattr from files"
  homepage "https://github.com/jurek-zsl/antiQuarantine/"
  url "https://github.com/jurek-zsl/homebrew-antiQuarantine/releases/download/v1.1.0/aq"
  sha256 "13f443cce3ac1f17f5b71425745bdcb95c9ac2d246fedf043b9079bb6b1c0190"
  version "1.1.0"

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
