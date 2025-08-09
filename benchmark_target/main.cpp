#include <iostream>
#include <vector>
#include <cstdint>
#include <thread>
#include <chrono>
#include <unistd.h>
#include <sys/types.h>
#include <cstring>
#include <cstdlib>
#include <iomanip>

// A simple Linear Congruential Generator for deterministic pseudo-random numbers.
class LCG {
public:
    LCG(uint64_t seed) : current_seed(seed) {}

    uint8_t next() {
        current_seed = (a * current_seed + c); // No modulo to keep it fast, overflow is fine.
        return static_cast<uint8_t>(current_seed >> 32);
    }

private:
    uint64_t current_seed;
    static const uint64_t a = 1664525;
    static const uint64_t c = 1013904223;
};

void print_hex(const uint8_t* data, size_t len) {
    for (size_t i = 0; i < len; ++i) {
        std::cout << std::hex << std::setw(2) << std::setfill('0') << static_cast<int>(data[i]);
    }
}

int main() {
    // 1. Get memory size from environment variable.
    const char* mem_size_env = std::getenv("ALLOCATE_MEM_MB");
    size_t mem_size_mb = 4096; // Default to 4GiB
    if (mem_size_env) {
        mem_size_mb = std::stoul(mem_size_env);
    }
    const size_t mem_size = mem_size_mb * 1024 * 1024;

    // 2. Allocate memory.
    char* memory_block = new (std::nothrow) char[mem_size];
    if (!memory_block) {
        std::cerr << "Failed to allocate memory." << std::endl;
        return 1;
    }

    // 3. Fill with a deterministic byte sequence.
    LCG lcg(0xDEADBEEF); // Fixed seed for determinism
    for (size_t i = 0; i < mem_size; ++i) {
        memory_block[i] = lcg.next();
    }

    // 4. Generate and place three 32-byte patterns.
    const size_t pattern_size = 32;
    uint8_t pattern1[pattern_size];
    uint8_t pattern2[pattern_size];
    uint8_t pattern3[pattern_size];

    LCG pattern_lcg(0xCAFEFEED);
    for(size_t i = 0; i < pattern_size; ++i) pattern1[i] = pattern_lcg.next();
    for(size_t i = 0; i < pattern_size; ++i) pattern2[i] = pattern_lcg.next();
    for(size_t i = 0; i < pattern_size; ++i) pattern3[i] = pattern_lcg.next();

    size_t offset1 = mem_size / 4;
    size_t offset2 = mem_size / 2;
    size_t offset3 = (mem_size / 4) * 3;

    std::memcpy(memory_block + offset1, pattern1, pattern_size);
    std::memcpy(memory_block + offset2, pattern2, pattern_size);
    std::memcpy(memory_block + offset3, pattern3, pattern_size);

    // 5. Print PID, base address, and pattern info.
    pid_t pid = getpid();
    uintptr_t base_address = reinterpret_cast<uintptr_t>(memory_block);

    std::cout << "PID: " << pid << std::endl;
    std::cout << "BASE_ADDRESS: " << std::hex << base_address << std::endl;
    
    std::cout << "PATTERN_1: ";
    print_hex(pattern1, pattern_size);
    std::cout << " @ " << std::hex << (base_address + offset1) << std::endl;

    std::cout << "PATTERN_2: ";
    print_hex(pattern2, pattern_size);
    std::cout << " @ " << std::hex << (base_address + offset2) << std::endl;

    std::cout << "PATTERN_3: ";
    print_hex(pattern3, pattern_size);
    std::cout << " @ " << std::hex << (base_address + offset3) << std::endl;
    
    // 6. Wait for a signal from the parent process (a newline on stdin).
    std::cout << "READY" << std::endl;
    std::string line;
    std::getline(std::cin, line);

    delete[] memory_block;
    return 0;
}
