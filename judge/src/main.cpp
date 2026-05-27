#include <nlohmann/json.hpp>

#include <algorithm>
#include <array>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <sstream>
#include <string>
#include <vector>

#ifdef __linux__
#include <sys/resource.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>
#endif

using json = nlohmann::json;

struct TestCase {
    std::string input;
    std::string expected;
};

struct JudgeInput {
    std::string language;
    std::string code;
    std::vector<TestCase> testCases;
    int timeLimitMs  = 2000;
    int memLimitMb   = 256;
};

static void respondAndExit(const std::string& verdict) {
    std::cout << json{{"verdict", verdict}}.dump() << std::endl;
    std::exit(0);
}

// Run a binary (or interpreter + script) against one test case.
// Returns {stdout_output, timed_out}.
static std::pair<std::string, bool> runOne(
    const std::string& exe,
    const std::vector<std::string>& args,
    const std::string& input,
    int timeLimitMs,
    int memLimitMb)
{
    std::string output;
    bool timedOut = false;

#ifdef __linux__
    int inPipe[2], outPipe[2];
    if (pipe(inPipe) || pipe(outPipe)) return {"", false};

    pid_t pid = fork();
    if (pid == 0) {
        // child
        dup2(inPipe[0],  STDIN_FILENO);
        dup2(outPipe[1], STDOUT_FILENO);
        close(inPipe[1]); close(outPipe[0]);
        close(inPipe[0]); close(outPipe[1]);

        struct rlimit rl{};
        // CPU time limit (seconds, rounded up)
        rl.rlim_cur = rl.rlim_max = static_cast<rlim_t>((timeLimitMs + 999) / 1000 + 1);
        setrlimit(RLIMIT_CPU, &rl);
        // Virtual memory limit
        rl.rlim_cur = rl.rlim_max = static_cast<rlim_t>(memLimitMb) * 1024 * 1024 * 2;
        setrlimit(RLIMIT_AS, &rl);

        std::vector<const char*> argv;
        argv.push_back(exe.c_str());
        for (auto& a : args) argv.push_back(a.c_str());
        argv.push_back(nullptr);

        execvp(exe.c_str(), const_cast<char* const*>(argv.data()));
        _exit(1);
    }

    // parent
    close(inPipe[0]); close(outPipe[1]);

    // write input
    if (!input.empty()) write(inPipe[1], input.c_str(), input.size());
    close(inPipe[1]);

    // read output (cap at 64 KB)
    char buf[4096];
    ssize_t n;
    while ((n = read(outPipe[0], buf, sizeof(buf))) > 0) {
        output.append(buf, n);
        if (output.size() > 65536) break;
    }
    close(outPipe[0]);

    // wait with wall-clock timeout
    int status = 0;
    int elapsed = 0;
    while (elapsed < timeLimitMs + 500) {
        pid_t r = waitpid(pid, &status, WNOHANG);
        if (r == pid) break;
        usleep(10000); // 10 ms
        elapsed += 10;
    }
    if (waitpid(pid, &status, WNOHANG) != pid) {
        kill(pid, SIGKILL);
        waitpid(pid, &status, 0);
        timedOut = true;
    }
#else
    // Fallback (Windows / non-Linux): popen — no rlimit support
    std::string cmd = exe;
    for (auto& a : args) cmd += " " + a;
    FILE* pipe = popen(cmd.c_str(), "r");
    if (!pipe) return {"", false};
    std::array<char, 4096> buf{};
    while (fgets(buf.data(), buf.size(), pipe)) output += buf.data();
    pclose(pipe);
#endif

    return {output, timedOut};
}

int main() {
    // Read all of stdin as JSON
    std::string rawInput(std::istreambuf_iterator<char>(std::cin), {});

    JudgeInput judgeIn;
    try {
        auto j = json::parse(rawInput);
        judgeIn.language    = j.value("language", "cpp");
        judgeIn.code        = j.value("code", "");
        judgeIn.timeLimitMs = j.value("time_limit_ms", 2000);
        judgeIn.memLimitMb  = j.value("memory_limit_mb", 256);
        for (auto& tc : j.at("test_cases")) {
            judgeIn.testCases.push_back({tc.value("input", ""), tc.value("expected", "")});
        }
    } catch (const std::exception& e) {
        std::cerr << "parse error: " << e.what() << "\n";
        respondAndExit("judge_error");
    }

    if (judgeIn.code.empty() || judgeIn.testCases.empty()) {
        respondAndExit("judge_error");
    }

    // Normalise language
    std::string lang = judgeIn.language;
    std::transform(lang.begin(), lang.end(), lang.begin(), ::tolower);
    if (lang == "c++") lang = "cpp";

    std::string exePath;
    std::vector<std::string> runArgs;

    namespace fs = std::filesystem;
    fs::path tmpDir = fs::temp_directory_path() / ("judge_" + std::to_string(getpid()));
    fs::create_directories(tmpDir);

    if (lang == "cpp") {
        fs::path srcPath = tmpDir / "submission.cpp";
        fs::path binPath = tmpDir / "submission.out";

        std::ofstream src(srcPath);
        src << judgeIn.code;
        src.close();

        std::string compileCmd = "g++ -O2 -std=c++17 \"" + srcPath.string() +
                                 "\" -o \"" + binPath.string() + "\" 2>/dev/null";
        if (std::system(compileCmd.c_str()) != 0) {
            respondAndExit("compile_error");
        }
        exePath = binPath.string();

    } else if (lang == "python" || lang == "python3") {
        fs::path srcPath = tmpDir / "submission.py";
        std::ofstream src(srcPath);
        src << judgeIn.code;
        src.close();
        exePath = "python3";
        runArgs  = {srcPath.string()};

    } else {
        respondAndExit("unsupported_language");
    }

    // Run against each test case
    for (size_t i = 0; i < judgeIn.testCases.size(); ++i) {
        const auto& tc = judgeIn.testCases[i];
        auto [output, timedOut] = runOne(exePath, runArgs, tc.input,
                                         judgeIn.timeLimitMs, judgeIn.memLimitMb);
        if (timedOut) {
            respondAndExit("time_limit_exceeded");
        }
        if (output != tc.expected) {
            respondAndExit("wrong_answer");
        }
    }

    respondAndExit("accepted");
}
