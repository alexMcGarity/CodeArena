cmake_minimum_required(VERSION 3.20)
project(CodeArenaJudge LANGUAGES CXX)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

include(FetchContent)
FetchContent_Declare(
  nlohmann_json
  GIT_REPOSITORY https://github.com/nlohmann/json.git
  GIT_TAG        v3.11.3
  GIT_SHALLOW    TRUE
)
set(JSON_BuildTests OFF CACHE BOOL "" FORCE)
set(JSON_Install    OFF CACHE BOOL "" FORCE)
FetchContent_MakeAvailable(nlohmann_json)

add_executable(codearena-judge src/main.cpp)
target_link_libraries(codearena-judge PRIVATE nlohmann_json::nlohmann_json)
target_compile_options(codearena-judge PRIVATE -O2)
