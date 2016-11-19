#include <iostream>

#include <test.h>
#include <data.h>

extern void put();

int
main()
{
  Test t;
  std::cout << t.get() << std::endl;
  std::string a = "OK,Stop";
  Data d(a);
  std::cout << d.get() << std::endl;
  put();
  return 0;
}
//
