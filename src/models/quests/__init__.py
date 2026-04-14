from src.models.Quest import QuestRegistry
from .TutorialQuest import TutorialQuest
from .Day1StrataGreetingQuest import Day1StrataGreetingQuest
from .Day2AlchemyDecayQuest import Day2AlchemyDecayQuest
from .Day3EstablishingBaseQuest import Day3EstablishingBaseQuest
from .Day4WillToLifeQuest import Day4WillToLifeQuest
from .Day5WisdomOddsQuest import Day5WisdomOddsQuest
from .Day6GreatContributionQuest import Day6GreatContributionQuest
from .Day7FirstSproutQuest import Day7FirstSproutQuest

def initialize_quests():
    QuestRegistry.register(TutorialQuest())
    QuestRegistry.register(Day1StrataGreetingQuest())
    QuestRegistry.register(Day2AlchemyDecayQuest())
    QuestRegistry.register(Day3EstablishingBaseQuest())
    QuestRegistry.register(Day4WillToLifeQuest())
    QuestRegistry.register(Day5WisdomOddsQuest())
    QuestRegistry.register(Day6GreatContributionQuest())
    QuestRegistry.register(Day7FirstSproutQuest())
    print("✅ Quest system initialized and quests registered.")
