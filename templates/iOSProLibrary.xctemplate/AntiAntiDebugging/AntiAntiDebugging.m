//
//  AntiAntiDebugging.m
//  AntiAntiDebugging
//
//  Created by TrialMacApp on 2025-02-27.
//

#import <Foundation/Foundation.h>
#import "fishhook.h"


typedef int (*ptrace_t)(int, pid_t, caddr_t, int);
typedef int (*syscall_t)(int, ...);

// raw function pointer
ptrace_t original_ptrace = NULL;
syscall_t original_syscall = NULL;


static int change_ptrace(int request, pid_t pid, caddr_t addr, int data) {
    if (request != 31) { return original_ptrace(request, pid, addr, data); }
    return 0;
}

static int change_syscall(int code, va_list args) {
    if (code == 26) {
        int request = va_arg(args, int);
        if (request == 31) {
            return 0;
        }
    }
    return original_syscall(code, args);
}


__attribute__((constructor)) static void AntiAntiDebug(void) {
    struct rebinding rebinding_ptrace[] = {
            { "ptrace", (void *)change_ptrace, (void *)&original_ptrace }
        };
    rebind_symbols(rebinding_ptrace, 1);
    
//    // The most common way is to use ptrace anti-debugging. If necessary, enable the following functions
//    struct rebinding rebinding_syscall[] = {
//            { "syscall", (void *)change_syscall, (void *)&original_syscall }
//        };
//    rebind_symbols(rebinding_syscall, 1);
}
