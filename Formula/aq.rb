class Aq < Formula
  desc "antiQuarantine — CLI to remove com.apple.quarantine xattr from files"
  homepage "https://github.com/jurek-zsl/antiQuarantine/"
  url "https://github.com/jurek-zsl/homebrew-antiQuarantine/releases/download/v1.2.0/aq"
  sha256 "c1ab8165ce4bfda0a8d51fb327b1b4830c5c6b116d5261191f093fe774274dce"
  version "1.2.0"

  def install
    bin.install "aq"
  end

  def caveats
    <<~EOS
      Thanks for installing aq!
      Example:
        aq MyApp.app
        aq -r MyApp.app
        aq -f ~/Downloads
    EOS
  end
end
