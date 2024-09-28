#include "testlib.h"

#include <bits/stdc++.h>

using namespace std;

#define ll long long

const int MAXT = 43200;

int normalise(int val) {
    if (val < 0) {
        val = -val ;
    }
    return val;
}

int reverse_distance(int val) {
    return 60 - val;
}

int calc_min_distance(int T) {
    T = T % MAXT;

    int hours = T / (12 * 60);
    int minutes = (T / 60) % 60;
    int seconds = T % 60;

    int distance1 = normalise(hours - minutes);
    int distance2 = normalise(hours - seconds);
    int distance3 = normalise(minutes - seconds);

    int rev_distance1 = reverse_distance(distance1);
    int rev_distance2 = reverse_distance(distance2);
    int rev_distance3 = reverse_distance(distance3);

    return min(
        min(distance1, rev_distance1),
        min(
            min(distance2, rev_distance2),
            min(distance3, rev_distance3)
        )
    );
}


const int MAX_LIMIT = 150;

void score_solution(int pans) {
    if (pans > MAX_LIMIT)
    {
        quitf(_wa, "Correct, too many guesses!");
    }
    else if (pans <= 7)
    {
        quitf(_ok, "answer %d\n", pans);
    }
    else
    {
        double x = pans;
        double first = 2.0 / sqrt(x + 2);
        double second = 3.0 / (x + 2);
        double points = (first + second);
        if (points <= 0 || points >= 1)
        {
            quitf(_fail, "Interactor point calculation assertion error");
        }
        quitp(points, "Correct, too many guesses!");
    }
}


int main(int argc, char ** argv)
try {
	registerInteraction(argc, argv);

	cout.exceptions(ios_base::badbit | ios_base::failbit);

#ifdef SIGPIPE
	if (signal(SIGPIPE, SIG_IGN) == SIG_ERR) {
		throw std::system_error(errno, std::system_category(), "signal");
	}
#endif

	int T = inf.readInt(0, MAXT - 1, "T");

	int cnt = 0;
	while(true) {
        int type = ouf.readInt(0, 1, "Output type");

        if (type == 0) 
        {
            cnt++;
            int P = ouf.readInt(0, MAXT - 1, "P");
            tout << 0 << " " << P << endl;
            int distance = calc_min_distance(T + P);
            cout << distance << endl;
            tout << distance << endl;
        }
        else if (type == 1)
        {
            int Tpart = ouf.readInt(0, MAXT - 1, "T");
            tout << 1 << " " << Tpart << endl;
            if (Tpart != T) {
                quitf(_wa, "Incorrect guess");
            }

// #ifdef cms
                score_solution(cnt);
// #else
//                 quitf(_ok, "Participant outputted guess in %d queries!", cnt);
// #endif
        }
        else
        {
            quitf(_fail, "Incorrect interactor");
        }
	}
} catch (const std::exception& e) {
	quit(_pe, e.what());
}
