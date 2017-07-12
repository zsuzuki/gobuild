#include <iostream>

#include <test.h>
#include <data.h>

extern void put();
#ifndef TEST3
extern "C" {
  void putHello();
};
#endif

int
main()
{
  std::cout << USERNAME << std::endl;
  Test t;
  std::cout << t.get() << std::endl;
  std::string a = "OK,Stop";
  Data d(a);
  std::cout << d.get() << std::endl;
  put();
#ifndef TEST3
  putHello();
#endif
  return 0;
}
//
