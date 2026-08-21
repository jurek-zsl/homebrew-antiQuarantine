class Aq < Formula
  desc "High-performance macOS Gatekeeper quarantine management CLI"
  homepage "https://github.com/jurek-zsl/homebrew-antiQuarantine"
  url "https://github.com/jurek-zsl/homebrew-antiQuarantine/archive/refs/tags/v2.0.0.tar.gz"
  sha256 "b205ef598d7069798d834d43cca59eeae9b1d9873776794ff34061522dec8f29"
  license "MIT"
  head "https://github.com/jurek-zsl/homebrew-antiQuarantine.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X antiQuarantine/internal/cli.Version=#{version}"), "./cmd/aq"
    generate_completions_from_executable(bin/"aq", "completion")
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
