from src.models.Quest import QuestRegistry
from .main.Day0WelcomeToHoakHaven import Day0WelcomeToHoakHavenQuest
from .main.Day1StrataGreetingQuest import Day1StrataGreetingQuest
from .main.Day2AlchemyDecayQuest import Day2AlchemyDecayQuest
from .main.Day3EstablishingBaseQuest import Day3EstablishingBaseQuest
from .main.Day4WillToLifeQuest import Day4WillToLifeQuest
from .main.Day5WisdomOddsQuest import Day5WisdomOddsQuest
from .main.Day6GreatContributionQuest import Day6GreatContributionQuest
from .main.Day7FirstSproutQuest import Day7FirstSproutQuest

def initialize_quests():
    QuestRegistry.register(Day0WelcomeToHoakHavenQuest())
    QuestRegistry.register(Day1StrataGreetingQuest())
    QuestRegistry.register(Day2AlchemyDecayQuest())
    QuestRegistry.register(Day3EstablishingBaseQuest())
    QuestRegistry.register(Day4WillToLifeQuest())
    QuestRegistry.register(Day5WisdomOddsQuest())
    QuestRegistry.register(Day6GreatContributionQuest())
    QuestRegistry.register(Day7FirstSproutQuest())
    print("✅ Quest system initialized and quests registered.")
