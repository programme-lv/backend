#include <bits/stdc++.h>
#define fi first
#define se second
#define pb push_back
using namespace std;
int main()
{
    ios_base::sync_with_stdio(0);
    cin.tie(0);cout.tie(0);
    //ifstream cin("in.in");
    int n;
    cin >> n;
    vector<pair<int,int> > con;
    stack<pair<int,int> > st;
    int a[n+1];
    a[0]=-1;
    st.push({0,0});
    int mxlen = -1, l = -1, r = -1;
    for(int i = 1;i<=n;i++)
    {
        cin >> a[i];
        int curmi = i, curmx = i;
        while(1)
        {
            int mipos = st.top().fi, mxpos = st.top().se;
            if(a[mipos]==a[curmx]+1)
            {
                con.pb({a[curmi],a[mipos]});
                curmx=mxpos;
            }
            else if(a[mxpos]==a[curmi]-1)
            {
                con.pb({a[mipos],a[curmi]});
                curmi=mipos;
            }
            else
                break;
            st.pop();
        }
        st.push({curmi,curmx});
        int nwlen = a[curmx]-a[curmi]+1;
        if(nwlen>mxlen)
        {
            mxlen=nwlen;
            l=a[curmi];
            r=a[curmx];
        }
    }
    vector<pair<int,int> > ans;
    for(auto x:con)
        if(l<=min(x.fi,x.se)&&max(x.fi,x.se)<=r)
            ans.pb(x);
    cout << ans.size()+1 << " " << ans.size() << "\n";
    for(auto x:ans)
        cout << x.fi << " " << x.se << "\n";
    return 0;
}
