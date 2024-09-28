#include <bits/stdc++.h>

using namespace std;

int main() {
    int n;
    cin >> n;
    int l = 1;
    int r = n;
    int mid;
    while(l < r){
        mid = (l + r) / 2;
        cout << mid << endl;
        int res;
        cin >> res;
        if(res == -1){
            r = mid - 1;
        }
        else if(res == +1){
            l = mid + 1;
        }
        else{
            return 0;
        }
    }
    cout << l << endl;
    int res;
    cin >> res;
    return 0;
}