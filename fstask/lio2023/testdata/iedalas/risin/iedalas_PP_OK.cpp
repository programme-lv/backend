#include <bits/stdc++.h>

using namespace std;

const int MAXT = 720;
int diff[MAXT];

int min_diff(int i) {
  int minutes = i % 60;
  int hours = i / 12;

  int a = abs(hours - minutes);
  return min(a, 60 - a);
}

int add_t(int a, int b) {
  a += b;
  if (a >= 720)
    a -= 720;
  return a;
}

void calc() {
  for (int i = 0; i < MAXT; i++) {
    diff[i] = min_diff(i);
  }
}

const int MAXD = 30;
int cnt[MAXD + 1];

int best_div(const vector<int> &possibilities) {
  int best_div = 0;
  int gr_size = possibilities.size();
  for (int i = 0; i < MAXT; i++) {
    for (auto t : possibilities) {
      cnt[min_diff(add_t(t, i))]++;
    }

    int maxgr = 0;
    for (int j = 0; j <= MAXD; j++) {
      maxgr = max(maxgr, cnt[j]);
      cnt[j] = 0;
    }

    if (maxgr < gr_size) {
      best_div = i;
      gr_size = maxgr;
    }
  }

  return best_div;
}

int main() {
  calc();

  vector<int> possibilities;
  possibilities.reserve(720);
  for (int i = 0; i < 720; i++)
    possibilities.push_back(i);

  int q = 0;
  while (possibilities.size() > 1) {
    int q = best_div(possibilities);

    assert(0 <= q && q < 720);
    cout << 0 << " " << q << endl;

    int d;
    cin >> d;

    possibilities.erase(
        remove_if(possibilities.begin(), possibilities.end(),
                  [=](auto elem) { return min_diff(add_t(elem, q)) != d; }),
        possibilities.end());
  }

  cout << 1 << " " << possibilities[0] << endl;
}
