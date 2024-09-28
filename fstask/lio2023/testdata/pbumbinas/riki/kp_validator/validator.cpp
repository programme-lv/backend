#include "testlib.h"
#include "examples.h"
#include "subtask1.h"
#include "subtask2.h"
#include "subtask3.h"
#include "subtask4.h"
#include "subtask5.h"
#include "subtask6.h"
#include "subtask7.h"
#include "subtask8.h"

int main(int argc, char* argv[]) {
    registerValidation(argc, argv);

    int group;
    try{group=stoi(validator.group());}
    catch(...){
        ensuref(false,"validator's group must be [0-8]");
    }
    switch (group) {
        case 0:
            examples::validate(); break;
        case 1:
            subtask1::validate(); break;
        case 2:
            subtask2::validate(); break;
        case 3:
            subtask3::validate(); break;
        case 4:
            subtask4::validate(); break;
        case 5:
            subtask5::validate(); break;
        case 6:
            subtask6::validate(); break;
        case 7:
            subtask7::validate(); break;
        case 8:
            subtask8::validate(); break;
    }
    inf.readEof();
}