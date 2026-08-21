class Aq < Formula
  desc "High-performance macOS Gatekeeper quarantine management CLI"
  homepage "https://github.com/jurek-zsl/homebrew-antiQuarantine"
  version "2.0.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jurek-zsl/homebrew-antiQuarantine/releases/download/v2.0.0/aq_2.0.0_darwin_arm64.tar.gz"
      sha256 "496ca31b6d836643efd557c259afd2463cacd7aa764c9b7903f252d0ff6dc981"
    else
      url "https://github.com/jurek-zsl/homebrew-antiQuarantine/releases/download/v2.0.0/aq_2.0.0_darwin_amd64.tar.gz"
      sha256 "658fb9d92d7312c891bebea4dfb8dd931f83851a33522a9ff996abd6c737decd"
    end
  end

  def install
    bin.install "aq"
  end

  def caveats
    <<~EOS
      Thanks for installing antiQuarantine (aq) v2.0!
      
      Quick Examples:
        aq check MyApp.app           # Check Gatekeeper status
        aq inspect --json MyApp.app  # View origin download URL and timestamp
        aq strip MyApp.app           # Strip quarantine attribute
        aq fix-app MyApp.app         # Sanitize nested bundles & ad-hoc codesign
        aq restore --last            # Undo last stripped attribute
        aq watch ~/Downloads         # Background folder monitor daemon
        aq tui                       # Interactive terminal visual browser
    EOS
  end

  test do
    assert_match "aq", shell_output("#{bin}/aq --version")

    test_file = testpath/"sample_quarantine.txt"
    touch test_file
    system "/usr/bin/xattr", "-w", "com.apple.quarantine", "0081;65d8ab12;Safari;B8C27D56-5B81-4C3D-B9AC-06D76D38B1C8", test_file
    assert_match "HAS com.apple.quarantine", shell_output("#{bin}/aq check #{test_file}")
    system "#{bin}/aq", "strip", test_file
    assert_match "does NOT have com.apple.quarantine", shell_output("#{bin}/aq check #{test_file}")
  end
end
