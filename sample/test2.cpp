#include <iostream>
#if TEST3
#include <data.h>
#endif

int
main()
{
#if TEST3
  std::cout << "Hello,World(TEST3)" << std::endl;
  std::string a = "MyTest";
  Data d(a);
  std::cout << d.get() << std::endl;
#else
  std::cout << "Hello,World" << std::endl;
#endif
  return 0;
}
//
